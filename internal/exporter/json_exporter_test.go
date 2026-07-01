package exporter_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reporting-service/internal/domain"
	"reporting-service/internal/exporter"
)

func TestJSONExporter_Export(t *testing.T) {
	e := exporter.NewJSONExporter()
	data, err := e.Export(sampleResult())
	require.NoError(t, err)

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(data, &rows))
	require.Len(t, rows, 2)
	assert.Equal(t, "Asha", rows[0]["name"])

	assert.Equal(t, domain.FormatJSON, e.Format())
	assert.Equal(t, "application/json", e.ContentType())
}
