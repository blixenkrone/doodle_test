package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/blixenkrone/doodle/internal/onboarding"
	"github.com/blixenkrone/doodle/internal/timeslots"
	"github.com/blixenkrone/sdk/logger"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// newAPI wires the real handlers against a real migrated DB and returns an
// httptest server exercising the full HTTP path.
func newAPI(t *testing.T) *httptest.Server {
	t.Helper()
	f := setup(t)
	log := logger.New()

	onboardingH := onboarding.NewHTTPHandler(log, f.store)
	timeslotsH := timeslots.NewHTTPHandler(log, f.store)

	r := mux.NewRouter()
	r.HandleFunc("/users", onboardingH.CreateUser()).Methods(http.MethodPost)
	r.HandleFunc("/timeslots", timeslotsH.CreateTimeslot()).Methods(http.MethodPost)
	r.HandleFunc("/timeslots/allotted", timeslotsH.GetAllotted()).Methods(http.MethodGet)
	r.HandleFunc("/timeslots/calendar", timeslotsH.GetCalendar()).Methods(http.MethodGet)
	r.HandleFunc("/timeslots/meeting", timeslotsH.CreateMeeting()).Methods(http.MethodPost)
	r.HandleFunc("/timeslots/{id}", timeslotsH.UpdateTimeslot()).Methods(http.MethodPatch)
	r.HandleFunc("/timeslots/{id}", timeslotsH.DeleteTimeslot()).Methods(http.MethodDelete)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&v))
	return v
}

// TestAPIFullFlow walks the documented user journey end to end:
// create user -> create availability -> list allotted -> book meeting ->
// see it as booked and on the calendar.
func TestAPIFullFlow(t *testing.T) {
	api := newAPI(t)

	// 1. create user
	resp := postJSON(t, api.URL+"/users", map[string]any{"name": "alice"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	userID := decode[struct {
		ID string `json:"id"`
	}](t, resp).ID
	require.NotEmpty(t, userID)

	// 2. create availability (60 min window, 30 min slots -> 2 slots)
	start := time.Now().Add(time.Hour).UTC().Truncate(time.Minute)
	resp = postJSON(t, api.URL+"/timeslots", map[string]any{
		"user_id":       userID,
		"duration_mins": 30,
		"availability": []map[string]any{
			{"availability_time_start": start, "availability_time_to": start.Add(60 * time.Minute)},
		},
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	ids := decode[timeslots.CreateTimeslotResponse](t, resp).TimeslotIDs
	require.Len(t, ids, 2)

	// 3. list allotted
	allottedURL := api.URL + "/timeslots/allotted?user_id=" + userID +
		"&date_start=" + start.Add(-time.Hour).Format(time.RFC3339) +
		"&date_end=" + start.Add(24*time.Hour).Format(time.RFC3339)
	resp, err := http.Get(allottedURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	allotted := decode[[]timeslots.AllottedTimeslot](t, resp)
	require.Len(t, allotted, 2)
	require.False(t, allotted[0].IsBooked)

	// 4. book a meeting on the first slot
	resp = postJSON(t, api.URL+"/timeslots/meeting", map[string]any{
		"id":        ids[0],
		"title":     "Standup",
		"descr":     "daily",
		"url":       "http://meet/x",
		"attendees": []string{"a@x.com", "b@x.com"},
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// 5. double booking the same slot -> 409
	resp = postJSON(t, api.URL+"/timeslots/meeting", map[string]any{"id": ids[0], "title": "dup"})
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()

	// 6. deleting a booked slot -> 409
	req, _ := http.NewRequest(http.MethodDelete, api.URL+"/timeslots/"+ids[0], nil)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()

	// 7. calendar shows the booked meeting with attendees
	resp, err = http.Get(api.URL + "/timeslots/calendar?user_id=" + userID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	cal := decode[[]timeslots.CalendarMeeting](t, resp)
	require.Len(t, cal, 1)
	require.Equal(t, "Standup", cal[0].Title)
	require.ElementsMatch(t, []string{"a@x.com", "b@x.com"}, cal[0].Attendees)
}
