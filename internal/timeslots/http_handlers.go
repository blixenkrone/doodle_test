// Package timeslots holds the HTTP handlers for timeslot availability
// management and meeting scheduling.
package timeslots

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/blixenkrone/doodle/internal/storage"
	sdkhttp "github.com/blixenkrone/sdk/http"
	"github.com/blixenkrone/sdk/logger"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// timeslotStore is the subset of the storage layer this package depends on.
type timeslotStore interface {
	CreateAvailability(ctx context.Context, userID uuid.UUID, windows []storage.AvailabilityWindow, durationMins int32) ([]storage.DoodleTimeslot, error)
	ListAllottedTimeslots(ctx context.Context, arg storage.ListAllottedTimeslotsParams) ([]storage.ListAllottedTimeslotsRow, error)
	UpdateTimeslot(ctx context.Context, arg storage.UpdateTimeslotParams) (int64, error)
	DeleteTimeslot(ctx context.Context, timeslotID uuid.UUID) (int64, error)
	BookMeeting(ctx context.Context, arg storage.CreateMeetingParams, attendees []string) (storage.DoodleMeeting, error)
	ListCalendarMeetings(ctx context.Context, userID uuid.UUID) ([]storage.ListCalendarMeetingsRow, error)
}

type HTTPHandler struct {
	logger logger.Logger
	store  timeslotStore
}

func NewHTTPHandler(logger logger.Logger, store timeslotStore) HTTPHandler {
	return HTTPHandler{logger, store}
}

const (
	minDurationMins = 10
	maxDurationMins = 60
)

// --- Create timeslots -------------------------------------------------------

type Availability struct {
	StartAt time.Time `json:"availability_time_start"`
	EndAt   time.Time `json:"availability_time_to"`
}

type CreateTimeslotRequest struct {
	UserID       string         `json:"user_id"`
	Availability []Availability `json:"availability"`
	DurationMins int32          `json:"duration_mins"`
}

type CreateTimeslotResponse struct {
	TimeslotIDs []string `json:"timeslot_ids"`
}

// CreateTimeslot creates availability windows, sliced into slots of duration_mins.
// @Summary Create available timeslots
// @Tags timeslots
// @Accept json
// @Produce json
// @Param body body CreateTimeslotRequest true "request body"
// @Success 201 {object} CreateTimeslotResponse
// @Failure 400 {object} sdkhttp.HTTPError
// @Failure 500 {object} sdkhttp.HTTPError
// @Router /timeslots [post]
func (h HTTPHandler) CreateTimeslot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := sdkhttp.DecodeJSON[CreateTimeslotRequest](r.Body)
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest)
			return
		}

		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
		if req.DurationMins < minDurationMins || req.DurationMins > maxDurationMins {
			sdkhttp.JSONError(w, http.StatusBadRequest, fmt.Errorf("duration_mins must be between %d and %d", minDurationMins, maxDurationMins))
			return
		}
		if len(req.Availability) == 0 {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("availability is required"))
			return
		}

		windows := make([]storage.AvailabilityWindow, 0, len(req.Availability))
		for _, a := range req.Availability {
			if !a.EndAt.After(a.StartAt) {
				sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("availability_time_to must be after availability_time_start"))
				return
			}
			windows = append(windows, storage.AvailabilityWindow{StartAt: a.StartAt, EndAt: a.EndAt})
		}

		created, err := h.store.CreateAvailability(r.Context(), userID, windows, req.DurationMins)
		if err != nil {
			h.logger.WithContext(r.Context()).Errorf("create availability: %v", err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}

		ids := make([]string, len(created))
		for i, ts := range created {
			ids[i] = ts.TimeslotID.String()
		}
		if err := sdkhttp.EncodeJSON(w, http.StatusCreated, CreateTimeslotResponse{TimeslotIDs: ids}); err != nil {
			h.logger.Errorln(err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
		}
	}
}

type AllottedTimeslot struct {
	ID           string    `json:"id"`
	TimeStart    time.Time `json:"time_start"`
	TimeEnd      time.Time `json:"time_end"`
	DurationMins int32     `json:"duration_mins"`
	IsBooked     bool      `json:"is_booked"`
}

// GetAllotted lists a user's upcoming timeslots within a time window.
// @Summary Get allotted timeslots
// @Tags timeslots
// @Produce json
// @Param user_id query string true "user id"
// @Param date_start query string true "RFC3339 window start"
// @Param date_end query string true "RFC3339 window end"
// @Success 200 {array} AllottedTimeslot
// @Failure 400 {object} sdkhttp.HTTPError
// @Failure 500 {object} sdkhttp.HTTPError
// @Router /timeslots/allotted [get]
func (h HTTPHandler) GetAllotted() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		userID, err := uuid.Parse(q.Get("user_id"))
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
		start, err := time.Parse(time.RFC3339, q.Get("date_start"))
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("invalid date_start (want RFC3339)"))
			return
		}
		end, err := time.Parse(time.RFC3339, q.Get("date_end"))
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("invalid date_end (want RFC3339)"))
			return
		}

		rows, err := h.store.ListAllottedTimeslots(r.Context(), storage.ListAllottedTimeslotsParams{
			UserID:    userID,
			DateStart: start,
			DateEnd:   end,
			Now:       time.Now(),
		})
		if err != nil {
			h.logger.WithContext(r.Context()).Errorf("list allotted: %v", err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}

		out := make([]AllottedTimeslot, len(rows))
		for i, row := range rows {
			out[i] = AllottedTimeslot{
				ID:           row.TimeslotID.String(),
				TimeStart:    row.StartAt,
				TimeEnd:      row.EndAt,
				DurationMins: row.DurationMins,
				IsBooked:     row.IsBooked,
			}
		}
		if err := sdkhttp.EncodeJSON(w, http.StatusOK, out); err != nil {
			h.logger.Errorln(err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
		}
	}
}

type UpdateTimeslotRequest struct {
	StartAt      time.Time `json:"time_start"`
	EndAt        time.Time `json:"time_end"`
	DurationMins int32     `json:"duration_mins"`
}

// UpdateTimeslot mutates an unbooked timeslot. Returns 409 if a meeting is booked.
// @Summary Update a timeslot
// @Tags timeslots
// @Accept json
// @Produce json
// @Param id path string true "timeslot id"
// @Param body body UpdateTimeslotRequest true "request body"
// @Success 204
// @Failure 400 {object} sdkhttp.HTTPError
// @Failure 409 {object} sdkhttp.HTTPError
// @Failure 500 {object} sdkhttp.HTTPError
// @Router /timeslots/{id} [patch]
func (h HTTPHandler) UpdateTimeslot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(mux.Vars(r)["id"])
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("invalid timeslot id"))
			return
		}
		req, err := sdkhttp.DecodeJSON[UpdateTimeslotRequest](r.Body)
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest)
			return
		}
		if !req.EndAt.After(req.StartAt) {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("time_end must be after time_start"))
			return
		}
		if req.DurationMins < minDurationMins || req.DurationMins > maxDurationMins {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("duration_mins must be between 10 and 60"))
			return
		}

		rows, err := h.store.UpdateTimeslot(r.Context(), storage.UpdateTimeslotParams{
			TimeslotID:   id,
			StartAt:      req.StartAt,
			EndAt:        req.EndAt,
			DurationMins: req.DurationMins,
		})
		if err != nil {
			h.logger.WithContext(r.Context()).Errorf("update timeslot: %v", err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}
		if rows == 0 {
			sdkhttp.JSONError(w, http.StatusConflict, errors.New("timeslot not found or already booked"))
			return
		}
		sdkhttp.JSONError(w, http.StatusNoContent)
	}
}

// DeleteTimeslot removes an unbooked timeslot. Returns 409 if a meeting is booked.
// @Summary Delete a timeslot
// @Tags timeslots
// @Produce json
// @Param id path string true "timeslot id"
// @Success 204
// @Failure 400 {object} sdkhttp.HTTPError
// @Failure 409 {object} sdkhttp.HTTPError
// @Failure 500 {object} sdkhttp.HTTPError
// @Router /timeslots/{id} [delete]
func (h HTTPHandler) DeleteTimeslot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(mux.Vars(r)["id"])
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("invalid timeslot id"))
			return
		}
		rows, err := h.store.DeleteTimeslot(r.Context(), id)
		if err != nil {
			h.logger.WithContext(r.Context()).Errorf("delete timeslot: %v", err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}
		if rows == 0 {
			sdkhttp.JSONError(w, http.StatusConflict, errors.New("timeslot not found or already booked"))
			return
		}
		sdkhttp.JSONError(w, http.StatusNoContent)
	}
}

type CreateMeetingRequest struct {
	TimeslotID  string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"descr"`
	Attendees   []string `json:"attendees"`
	URL         string   `json:"url"`
}

type CreateMeetingResponse struct {
	MeetingID string `json:"meeting_id"`
}

// CreateMeeting books a timeslot as a meeting.
// @Summary Book a meeting on a timeslot
// @Tags meetings
// @Accept json
// @Produce json
// @Param body body CreateMeetingRequest true "request body"
// @Success 201 {object} CreateMeetingResponse
// @Failure 400 {object} sdkhttp.HTTPError
// @Failure 409 {object} sdkhttp.HTTPError
// @Failure 500 {object} sdkhttp.HTTPError
// @Router /timeslots/meeting [post]
func (h HTTPHandler) CreateMeeting() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := sdkhttp.DecodeJSON[CreateMeetingRequest](r.Body)
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest)
			return
		}
		timeslotID, err := uuid.Parse(req.TimeslotID)
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("invalid timeslot id"))
			return
		}
		if req.Title == "" {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("title is required"))
			return
		}

		meeting, err := h.store.BookMeeting(r.Context(), storage.CreateMeetingParams{
			MeetingID:   uuid.New(),
			TimeslotID:  timeslotID,
			Title:       req.Title,
			Description: req.Description,
			Url:         req.URL,
		}, req.Attendees)
		if err != nil {
			if errors.Is(err, storage.ErrSlotTaken) {
				sdkhttp.JSONError(w, http.StatusConflict, errors.New("timeslot already booked"))
				return
			}
			h.logger.WithContext(r.Context()).Errorf("book meeting: %v", err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}
		if err := sdkhttp.EncodeJSON(w, http.StatusCreated, CreateMeetingResponse{MeetingID: meeting.MeetingID.String()}); err != nil {
			h.logger.Errorln(err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
		}
	}
}

type CalendarMeeting struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	DatetimeStart time.Time `json:"datetime_start"`
	DatetimeEnd   time.Time `json:"datetime_end"`
	Attendees     []string  `json:"attendees"`
}

// GetCalendar returns the booked meetings on a user's personal calendar.
// @Summary See personal calendar
// @Tags meetings
// @Produce json
// @Param user_id query string true "user id"
// @Success 200 {array} CalendarMeeting
// @Failure 400 {object} sdkhttp.HTTPError
// @Failure 500 {object} sdkhttp.HTTPError
// @Router /timeslots/calendar [get]
func (h HTTPHandler) GetCalendar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(r.URL.Query().Get("user_id"))
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
		rows, err := h.store.ListCalendarMeetings(r.Context(), userID)
		if err != nil {
			h.logger.WithContext(r.Context()).Errorf("list calendar: %v", err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}
		out := make([]CalendarMeeting, len(rows))
		for i, row := range rows {
			out[i] = CalendarMeeting{
				ID:            row.MeetingID.String(),
				Title:         row.Title,
				DatetimeStart: row.StartAt,
				DatetimeEnd:   row.EndAt,
				Attendees:     row.Attendees,
			}
		}
		if err := sdkhttp.EncodeJSON(w, http.StatusOK, out); err != nil {
			h.logger.Errorln(err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
		}
	}
}
