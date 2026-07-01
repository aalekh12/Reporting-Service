package response_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"reporting-service/internal/domain"
	"reporting-service/pkg/response"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	response.JSON(w, 201, map[string]string{"foo": "bar"})
	assert.Equal(t, 201, w.Code)
	assert.Contains(t, w.Body.String(), `"foo":"bar"`)
}

func TestError_ValidationMapsTo400(t *testing.T) {
	w := httptest.NewRecorder()
	response.Error(w, domain.NewValidationError("bad input"))
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

func TestError_NotFoundMapsTo404(t *testing.T) {
	w := httptest.NewRecorder()
	response.Error(w, domain.NewNotFoundError("missing"))
	assert.Equal(t, 404, w.Code)
}

func TestError_InternalMapsTo500(t *testing.T) {
	w := httptest.NewRecorder()
	response.Error(w, domain.WrapInternal(errors.New("db down"), "query failed"))
	assert.Equal(t, 500, w.Code)
}

func TestError_UnknownErrorMapsTo500(t *testing.T) {
	w := httptest.NewRecorder()
	response.Error(w, errors.New("something else"))
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "internal")
}
