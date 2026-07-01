package exporter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reporting-service/internal/domain"
	"reporting-service/internal/exporter"
)

func sampleResult() *domain.ReportResult {
	return &domain.ReportResult{
		Columns: []string{"id", "name", "department"},
		Rows: []map[string]any{
			{"id": int64(1), "name": "Asha", "department": "Engineering"},
			{"id": int64(2), "name": "Ben", "department": nil},
		},
	}
}

func TestCSVExporter_Export(t *testing.T) {
	e := exporter.NewCSVExporter()
	data, err := e.Export(sampleResult())
	require.NoError(t, err)

	want := "id,name,department\n1,Asha,Engineering\n2,Ben,\n"
	assert.Equal(t, want, string(data))
	assert.Equal(t, domain.FormatCSV, e.Format())
	assert.Equal(t, "text/csv", e.ContentType())
	assert.Equal(t, "csv", e.FileExtension())
}

func TestCSVExporter_EmptyRows(t *testing.T) {
	e := exporter.NewCSVExporter()
	data, err := e.Export(&domain.ReportResult{Columns: []string{"id"}})
	require.NoError(t, err)
	assert.Equal(t, "id\n", string(data))
}
