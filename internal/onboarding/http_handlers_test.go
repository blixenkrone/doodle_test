package onboarding

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blixenkrone/doodle/internal/storage"
	"github.com/blixenkrone/sdk/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	err error
}

func (m mockStore) CreateUser(ctx context.Context, arg storage.CreateUserParams) (storage.DoodleUser, error) {
	if m.err != nil {
		return storage.DoodleUser{}, m.err
	}
	return storage.DoodleUser{UserID: arg.UserID, Name: arg.Name}, nil
}

func newHandler(err error) HTTPHandler {
	return NewHTTPHandler(logger.New(), mockStore{err: err})
}

func TestCreateUser(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"alice"}`))
		newHandler(nil).CreateUser()(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), `"id"`)
	})

	t.Run("name too short -> 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"ab"}`))
		newHandler(nil).CreateUser()(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid name chars -> 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"bad@name!"}`))
		newHandler(nil).CreateUser()(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("malformed json -> 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{`))
		newHandler(nil).CreateUser()(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("store error -> 500", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"alice"}`))
		newHandler(errors.New("boom")).CreateUser()(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
