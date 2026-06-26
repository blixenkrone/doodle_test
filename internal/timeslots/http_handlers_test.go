package timeslots

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blixenkrone/doodle/internal/storage"
	"github.com/blixenkrone/sdk/logger"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore is a configurable fake implementing timeslotStore. Each func field,
// when set, overrides the default no-op behaviour.
type mockStore struct {
	createAvailability func(ctx context.Context, userID uuid.UUID, w []storage.AvailabilityWindow, d int32) ([]storage.DoodleTimeslot, error)
	listAllotted       func(ctx context.Context, arg storage.ListAllottedTimeslotsParams) ([]storage.ListAllottedTimeslotsRow, error)
	updateTimeslot     func(ctx context.Context, arg storage.UpdateTimeslotParams) (int64, error)
	deleteTimeslot     func(ctx context.Context, id uuid.UUID) (int64, error)
	bookMeeting        func(ctx context.Context, arg storage.CreateMeetingParams, att []string) (storage.DoodleMeeting, error)
	listCalendar       func(ctx context.Context, userID uuid.UUID) ([]storage.ListCalendarMeetingsRow, error)
}

func (m mockStore) CreateAvailability(ctx context.Context, userID uuid.UUID, w []storage.AvailabilityWindow, d int32) ([]storage.DoodleTimeslot, error) {
	return m.createAvailability(ctx, userID, w, d)
}
func (m mockStore) ListAllottedTimeslots(ctx context.Context, arg storage.ListAllottedTimeslotsParams) ([]storage.ListAllottedTimeslotsRow, error) {
	return m.listAllotted(ctx, arg)
}
func (m mockStore) UpdateTimeslot(ctx context.Context, arg storage.UpdateTimeslotParams) (int64, error) {
	return m.updateTimeslot(ctx, arg)
}
func (m mockStore) DeleteTimeslot(ctx context.Context, id uuid.UUID) (int64, error) {
	return m.deleteTimeslot(ctx, id)
}
func (m mockStore) BookMeeting(ctx context.Context, arg storage.CreateMeetingParams, att []string) (storage.DoodleMeeting, error) {
	return m.bookMeeting(ctx, arg, att)
}
func (m mockStore) ListCalendarMeetings(ctx context.Context, userID uuid.UUID) ([]storage.ListCalendarMeetingsRow, error) {
	return m.listCalendar(ctx, userID)
}

func handler(m mockStore) HTTPHandler { return NewHTTPHandler(logger.New(), m) }

func TestCreateTimeslot(t *testing.T) {
	uid := uuid.New().String()

	t.Run("happy path", func(t *testing.T) {
		m := mockStore{createAvailability: func(_ context.Context, _ uuid.UUID, w []storage.AvailabilityWindow, d int32) ([]storage.DoodleTimeslot, error) {
			require.Len(t, w, 1)
			require.EqualValues(t, 30, d)
			return []storage.DoodleTimeslot{{TimeslotID: uuid.New()}, {TimeslotID: uuid.New()}}, nil
		}}
		body := `{"user_id":"` + uid + `","duration_mins":30,"availability":[{"availability_time_start":"2026-07-01T09:00:00Z","availability_time_to":"2026-07-01T10:00:00Z"}]}`
		rec := httptest.NewRecorder()
		handler(m).CreateTimeslot()(rec, httptest.NewRequest(http.MethodPost, "/timeslots", strings.NewReader(body)))
		require.Equal(t, http.StatusCreated, rec.Code)

		var resp CreateTimeslotResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Len(t, resp.TimeslotIDs, 2)
	})

	t.Run("duration out of range -> 400", func(t *testing.T) {
		body := `{"user_id":"` + uid + `","duration_mins":90,"availability":[{"availability_time_start":"2026-07-01T09:00:00Z","availability_time_to":"2026-07-01T10:00:00Z"}]}`
		rec := httptest.NewRecorder()
		handler(mockStore{}).CreateTimeslot()(rec, httptest.NewRequest(http.MethodPost, "/timeslots", strings.NewReader(body)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid user_id -> 400", func(t *testing.T) {
		body := `{"user_id":"nope","duration_mins":30,"availability":[{"availability_time_start":"2026-07-01T09:00:00Z","availability_time_to":"2026-07-01T10:00:00Z"}]}`
		rec := httptest.NewRecorder()
		handler(mockStore{}).CreateTimeslot()(rec, httptest.NewRequest(http.MethodPost, "/timeslots", strings.NewReader(body)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("window end before start -> 400", func(t *testing.T) {
		body := `{"user_id":"` + uid + `","duration_mins":30,"availability":[{"availability_time_start":"2026-07-01T10:00:00Z","availability_time_to":"2026-07-01T09:00:00Z"}]}`
		rec := httptest.NewRecorder()
		handler(mockStore{}).CreateTimeslot()(rec, httptest.NewRequest(http.MethodPost, "/timeslots", strings.NewReader(body)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestGetAllotted(t *testing.T) {
	uid := uuid.New()

	t.Run("happy path", func(t *testing.T) {
		m := mockStore{listAllotted: func(_ context.Context, arg storage.ListAllottedTimeslotsParams) ([]storage.ListAllottedTimeslotsRow, error) {
			assert.Equal(t, uid, arg.UserID)
			return []storage.ListAllottedTimeslotsRow{{TimeslotID: uuid.New(), IsBooked: true}}, nil
		}}
		url := "/timeslots/allotted?user_id=" + uid.String() + "&date_start=2026-07-01T00:00:00Z&date_end=2026-07-02T00:00:00Z"
		rec := httptest.NewRecorder()
		handler(m).GetAllotted()(rec, httptest.NewRequest(http.MethodGet, url, nil))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"is_booked":true`)
	})

	t.Run("bad date_start -> 400", func(t *testing.T) {
		url := "/timeslots/allotted?user_id=" + uid.String() + "&date_start=oops&date_end=2026-07-02T00:00:00Z"
		rec := httptest.NewRecorder()
		handler(mockStore{}).GetAllotted()(rec, httptest.NewRequest(http.MethodGet, url, nil))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// withVars attaches mux route vars so handlers reading mux.Vars work in tests.
func withVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

func TestDeleteTimeslot(t *testing.T) {
	id := uuid.New()

	t.Run("deleted -> 204", func(t *testing.T) {
		m := mockStore{deleteTimeslot: func(_ context.Context, got uuid.UUID) (int64, error) {
			assert.Equal(t, id, got)
			return 1, nil
		}}
		rec := httptest.NewRecorder()
		req := withVars(httptest.NewRequest(http.MethodDelete, "/timeslots/"+id.String(), nil), map[string]string{"id": id.String()})
		handler(m).DeleteTimeslot()(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("booked or missing -> 409", func(t *testing.T) {
		m := mockStore{deleteTimeslot: func(_ context.Context, _ uuid.UUID) (int64, error) { return 0, nil }}
		rec := httptest.NewRecorder()
		req := withVars(httptest.NewRequest(http.MethodDelete, "/timeslots/"+id.String(), nil), map[string]string{"id": id.String()})
		handler(m).DeleteTimeslot()(rec, req)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("invalid id -> 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := withVars(httptest.NewRequest(http.MethodDelete, "/timeslots/nope", nil), map[string]string{"id": "nope"})
		handler(mockStore{}).DeleteTimeslot()(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestUpdateTimeslot(t *testing.T) {
	id := uuid.New()
	body := `{"time_start":"2026-07-01T09:00:00Z","time_end":"2026-07-01T09:30:00Z","duration_mins":30}`

	t.Run("updated -> 204", func(t *testing.T) {
		m := mockStore{updateTimeslot: func(_ context.Context, _ storage.UpdateTimeslotParams) (int64, error) { return 1, nil }}
		rec := httptest.NewRecorder()
		req := withVars(httptest.NewRequest(http.MethodPatch, "/timeslots/"+id.String(), strings.NewReader(body)), map[string]string{"id": id.String()})
		handler(m).UpdateTimeslot()(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("booked -> 409", func(t *testing.T) {
		m := mockStore{updateTimeslot: func(_ context.Context, _ storage.UpdateTimeslotParams) (int64, error) { return 0, nil }}
		rec := httptest.NewRecorder()
		req := withVars(httptest.NewRequest(http.MethodPatch, "/timeslots/"+id.String(), strings.NewReader(body)), map[string]string{"id": id.String()})
		handler(m).UpdateTimeslot()(rec, req)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestCreateMeeting(t *testing.T) {
	tsID := uuid.New().String()

	t.Run("happy path", func(t *testing.T) {
		m := mockStore{bookMeeting: func(_ context.Context, arg storage.CreateMeetingParams, att []string) (storage.DoodleMeeting, error) {
			assert.Equal(t, "Standup", arg.Title)
			assert.Equal(t, []string{"a@x.com"}, att)
			return storage.DoodleMeeting{MeetingID: arg.MeetingID}, nil
		}}
		body := `{"id":"` + tsID + `","title":"Standup","descr":"d","url":"http://x","attendees":["a@x.com"]}`
		rec := httptest.NewRecorder()
		handler(m).CreateMeeting()(rec, httptest.NewRequest(http.MethodPost, "/timeslots/meeting", strings.NewReader(body)))
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("slot taken -> 409", func(t *testing.T) {
		m := mockStore{bookMeeting: func(_ context.Context, _ storage.CreateMeetingParams, _ []string) (storage.DoodleMeeting, error) {
			return storage.DoodleMeeting{}, storage.ErrSlotTaken
		}}
		body := `{"id":"` + tsID + `","title":"Standup"}`
		rec := httptest.NewRecorder()
		handler(m).CreateMeeting()(rec, httptest.NewRequest(http.MethodPost, "/timeslots/meeting", strings.NewReader(body)))
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("missing title -> 400", func(t *testing.T) {
		body := `{"id":"` + tsID + `"}`
		rec := httptest.NewRecorder()
		handler(mockStore{}).CreateMeeting()(rec, httptest.NewRequest(http.MethodPost, "/timeslots/meeting", strings.NewReader(body)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("store error -> 500", func(t *testing.T) {
		m := mockStore{bookMeeting: func(_ context.Context, _ storage.CreateMeetingParams, _ []string) (storage.DoodleMeeting, error) {
			return storage.DoodleMeeting{}, errors.New("boom")
		}}
		body := `{"id":"` + tsID + `","title":"Standup"}`
		rec := httptest.NewRecorder()
		handler(m).CreateMeeting()(rec, httptest.NewRequest(http.MethodPost, "/timeslots/meeting", strings.NewReader(body)))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestGetCalendar(t *testing.T) {
	uid := uuid.New()
	m := mockStore{listCalendar: func(_ context.Context, got uuid.UUID) ([]storage.ListCalendarMeetingsRow, error) {
		assert.Equal(t, uid, got)
		return []storage.ListCalendarMeetingsRow{{
			MeetingID: uuid.New(), Title: "Standup",
			StartAt: time.Now(), EndAt: time.Now().Add(time.Hour),
			Attendees: []string{"a@x.com"},
		}}, nil
	}}
	rec := httptest.NewRecorder()
	handler(m).GetCalendar()(rec, httptest.NewRequest(http.MethodGet, "/timeslots/calendar?user_id="+uid.String(), nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"a@x.com"`)
}
