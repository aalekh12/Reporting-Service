package exporter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reporting-service/internal/domain"
	"reporting-service/internal/exporter"
)

func TestRegistry_GetKnownFormats(t *testing.T) {
	r := exporter.DefaultRegistry()

	for _, format := range []domain.ExportFormat{domain.FormatJSON, domain.FormatCSV, domain.FormatExcel} {
		e, err := r.Get(format)
		require.NoError(t, err)
		assert.Equal(t, format, e.Format())
	}
}

func TestRegistry_GetUnknownFormat(t *testing.T) {
	r := exporter.DefaultRegistry()
	_, err := r.Get("pdf")
	assert.Error(t, err)
}
