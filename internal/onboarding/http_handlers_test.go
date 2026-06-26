package onboarding

import (
	"testing"
)

func TestCreateOrganization(t *testing.T) {

	t.Run("happy path", func(t *testing.T) {
	})

	t.Run("missing slug -> 400", func(t *testing.T) {
	})

	t.Run("malformed json -> 400", func(t *testing.T) {
	})

	t.Run("store error -> 500", func(t *testing.T) {
	})
}

type mockStore struct {
	err error
}
