package domain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"reporting-service/internal/domain"
)

func TestNewValidationError(t *testing.T) {
	err := domain.NewValidationError("bad field %q", "status")
	assert.Equal(t, domain.KindValidation, err.Kind)
	assert.Equal(t, `bad field "status"`, err.Error())
}

func TestNewNotFoundError(t *testing.T) {
	err := domain.NewNotFoundError("report %d not found", 42)
	assert.Equal(t, domain.KindNotFound, err.Kind)
	assert.Contains(t, err.Error(), "42")
}

func TestWrapInternal(t *testing.T) {
	cause := errors.New("connection reset")
	err := domain.WrapInternal(cause, "query failed")
	assert.Equal(t, domain.KindInternal, err.Kind)
	assert.Contains(t, err.Error(), "connection reset")
	assert.ErrorIs(t, err, cause)
}
