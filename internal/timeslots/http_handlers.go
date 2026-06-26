package timeslots

import (
	"net/http"

	sdkhttp "github.com/blixenkrone/sdk/http"
	"github.com/blixenkrone/sdk/logger"
)

type timeslotStore interface {
}

type HTTPHandler struct {
	logger logger.Logger
	store  timeslotStore
}

func NewHTTPHandler(logger logger.Logger, store timeslotStore) HTTPHandler {
	return HTTPHandler{logger, store}
}

type CreateTimeslotRequest struct {
}
type CreateTimeslotResponse struct {
}

// CreateTimeslot
// @Tags timeslots
// @Accept json
// @Produce json
// @Param body body CreateUserRequest true "request body"
// @Success 201 {object}
// @Failure 400 {object} sdkhttp.HTTPError
// @Failure 500 {object} sdkhttp.HTTPError
// @Router /timeslots [post]
func (h HTTPHandler) CreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := sdkhttp.DecodeJSON[CreateTimeslotRequest](r.Body)
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest)
			return
		}
		_ = req

		resp := CreateTimeslotResponse{}
		if err := sdkhttp.EncodeJSON(w, http.StatusCreated, resp); err != nil {
			h.logger.Errorln(err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}
	}
}
