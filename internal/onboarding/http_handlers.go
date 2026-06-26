package onboarding

import (
	"context"
	"errors"
	"net/http"
	"regexp"

	"github.com/blixenkrone/doodle/internal/storage"
	sdkhttp "github.com/blixenkrone/sdk/http"
	"github.com/blixenkrone/sdk/logger"
	"github.com/google/uuid"
)

type onboardingStore interface {
	CreateUser(ctx context.Context, arg storage.CreateUserParams) (storage.DoodleUser, error)
}

type HTTPHandler struct {
	logger logger.Logger
	store  onboardingStore
}

func NewHTTPHandler(logger logger.Logger, store onboardingStore) HTTPHandler {
	return HTTPHandler{logger, store}
}

type CreateUserRequest struct {
	Name string `json:"slug"`
}

type CreateUserResponse struct {
	ID string `json:"id"`
}

// ASCII letters and digits
var userNameRequirementRegexp = regexp.MustCompile("/^[A-Za-z0-9]+(?:[ _-][A-Za-z0-9]+)*$/")

// CreateUser onboards a new tenant (hospital User).
// @Summary Create an User (tenant)
// @Tags onboarding
// @Accept json
// @Produce json
// @Param body body CreateUserRequest true "request body"
// @Success 201 {object} CreateUserResponse
// @Failure 400 {object} sdkhttp.HTTPError
// @Failure 500 {object} sdkhttp.HTTPError
// @Router /users [post]
func (h HTTPHandler) CreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := sdkhttp.DecodeJSON[CreateUserRequest](r.Body)
		if err != nil {
			sdkhttp.JSONError(w, http.StatusBadRequest)
			return
		}

		if len(req.Name) <= 3 {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("username requires 4 chars"))
			return
		}
		if !userNameRequirementRegexp.MatchString(req.Name) {
			sdkhttp.JSONError(w, http.StatusBadRequest, errors.New("bad username"))
			return
		}

		id := uuid.New()
		_, err = h.store.CreateUser(r.Context(), storage.CreateUserParams{
			UserID: id,
			Name:   req.Name,
		})
		if err != nil {
			h.logger.WithContext(r.Context()).Errorf("create organization: %v", err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}
		if err := sdkhttp.EncodeJSON(w, http.StatusCreated, CreateUserResponse{ID: id.String()}); err != nil {
			h.logger.Errorln(err)
			sdkhttp.MustJSONError(w, http.StatusInternalServerError)
			return
		}
	}
}
