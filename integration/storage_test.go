package integration

import (
	"sync"
	"testing"
	"time"

	"github.com/blixenkrone/doodle/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	store *storage.Store
	pool  *pgxpool.Pool
}

func setup(t *testing.T) fixture {
	t.Helper()
	pg := RunMigrations(t, t.Name())
	t.Cleanup(func() { _ = pg.Teardown() })
	pool := pg.Container()
	store := storage.NewStore(pool)
	return fixture{store: store, pool: pool}
}

func (f fixture) newUser(t *testing.T) uuid.UUID {
	t.Helper()
	u, err := f.store.CreateUser(t.Context(), storage.CreateUserParams{UserID: uuid.New(), Name: "dummy"})
	require.NoError(t, err)
	return u.UserID
}

func TestCreateUser(t *testing.T) {
	f := setup(t)
	usr, err := f.store.CreateUser(t.Context(), storage.CreateUserParams{
		UserID: uuid.New(),
		Name:   "dummy",
	})
	require.NoError(t, err)
	require.WithinDuration(t, usr.CreatedAt, time.Now(), time.Second)
}

func TestCreateAvailabilitySlicesWindow(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	uid := f.newUser(t)

	start := time.Now().Add(time.Hour).UTC().Truncate(time.Minute)
	slots, err := f.store.CreateAvailability(ctx, uid, []storage.AvailabilityWindow{
		{StartAt: start, EndAt: start.Add(60 * time.Minute)},
	}, 30)
	require.NoError(t, err)
	require.Len(t, slots, 2)
	assert.Equal(t, start, slots[0].StartAt.UTC())
	assert.Equal(t, start.Add(30*time.Minute), slots[0].EndAt.UTC())
	assert.Equal(t, start.Add(30*time.Minute), slots[1].StartAt.UTC())
}

func TestListAllottedFiltersPastAndFlagsBooked(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	uid := f.newUser(t)

	future := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Minute)
	slots, err := f.store.CreateAvailability(ctx, uid, []storage.AvailabilityWindow{
		{StartAt: future, EndAt: future.Add(60 * time.Minute)},
	}, 30)
	require.NoError(t, err)
	require.Len(t, slots, 2)

	_, err = f.store.CreateTimeslot(ctx, storage.CreateTimeslotParams{
		TimeslotID:   uuid.New(),
		UserID:       uid,
		StartAt:      time.Now().Add(-time.Hour),
		EndAt:        time.Now().Add(-30 * time.Minute),
		DurationMins: 30,
	})
	require.NoError(t, err)

	_, err = f.store.BookMeeting(ctx, storage.CreateMeetingParams{
		MeetingID:  uuid.New(),
		TimeslotID: slots[0].TimeslotID,
		Title:      "Standup",
	}, []string{"a@x.com"})
	require.NoError(t, err)

	rows, err := f.store.ListAllottedTimeslots(ctx, storage.ListAllottedTimeslotsParams{
		UserID:    uid,
		DateStart: time.Now().Add(-24 * time.Hour),
		DateEnd:   time.Now().Add(24 * time.Hour),
		Now:       time.Now(),
	})
	require.NoError(t, err)
	require.Len(t, rows, 2, "past slot must be excluded")

	booked := map[uuid.UUID]bool{}
	for _, r := range rows {
		booked[r.TimeslotID] = r.IsBooked
	}
	assert.True(t, booked[slots[0].TimeslotID])
	assert.False(t, booked[slots[1].TimeslotID])
}

func TestBookMeetingConcurrentOnlyOneWins(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	uid := f.newUser(t)

	future := time.Now().Add(time.Hour)
	slots, err := f.store.CreateAvailability(ctx, uid, []storage.AvailabilityWindow{
		{StartAt: future, EndAt: future.Add(30 * time.Minute)},
	}, 30)
	require.NoError(t, err)
	require.Len(t, slots, 1)
	slotID := slots[0].TimeslotID

	const racers = 8
	var wg sync.WaitGroup
	var ok, taken int
	wg.Add(racers)
	for range racers {
		go func() {
			defer wg.Done()
			_, err := f.store.BookMeeting(ctx, storage.CreateMeetingParams{
				MeetingID:  uuid.New(),
				TimeslotID: slotID,
				Title:      "race",
			}, nil)
			switch {
			case err == nil:
				ok++
			case assert.ErrorIs(t, err, storage.ErrSlotTaken):
				taken++
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, ok, "exactly one booking should succeed")
	assert.Equal(t, racers-1, taken, "all others should get ErrSlotTaken")
}

func TestDeleteAndUpdateBlockedWhenBooked(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	uid := f.newUser(t)

	future := time.Now().Add(time.Hour)
	slots, err := f.store.CreateAvailability(ctx, uid, []storage.AvailabilityWindow{
		{StartAt: future, EndAt: future.Add(30 * time.Minute)},
	}, 30)
	require.NoError(t, err)
	slotID := slots[0].TimeslotID

	rows, err := f.store.UpdateTimeslot(ctx, storage.UpdateTimeslotParams{
		TimeslotID: slotID, StartAt: future, EndAt: future.Add(20 * time.Minute), DurationMins: 20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)

	_, err = f.store.BookMeeting(ctx, storage.CreateMeetingParams{
		MeetingID: uuid.New(), TimeslotID: slotID, Title: "Standup",
	}, nil)
	require.NoError(t, err)

	rows, err = f.store.UpdateTimeslot(ctx, storage.UpdateTimeslotParams{
		TimeslotID: slotID, StartAt: future, EndAt: future.Add(15 * time.Minute), DurationMins: 15,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, rows, "update must be blocked when booked")

	rows, err = f.store.DeleteTimeslot(ctx, slotID)
	require.NoError(t, err)
	assert.EqualValues(t, 0, rows, "delete must be blocked when booked")
}

func TestListCalendarReturnsAttendees(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	uid := f.newUser(t)

	future := time.Now().Add(time.Hour)
	slots, err := f.store.CreateAvailability(ctx, uid, []storage.AvailabilityWindow{
		{StartAt: future, EndAt: future.Add(30 * time.Minute)},
	}, 30)
	require.NoError(t, err)

	_, err = f.store.BookMeeting(ctx, storage.CreateMeetingParams{
		MeetingID: uuid.New(), TimeslotID: slots[0].TimeslotID, Title: "Standup",
	}, []string{"a@x.com", "b@x.com"})
	require.NoError(t, err)

	cal, err := f.store.ListCalendarMeetings(ctx, uid)
	require.NoError(t, err)
	require.Len(t, cal, 1)
	assert.Equal(t, "Standup", cal[0].Title)
	assert.ElementsMatch(t, []string{"a@x.com", "b@x.com"}, cal[0].Attendees)
}
