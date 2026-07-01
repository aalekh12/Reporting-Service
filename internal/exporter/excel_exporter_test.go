package exporter_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"reporting-service/internal/domain"
	"reporting-service/internal/exporter"
)

func TestExcelExporter_Export(t *testing.T) {
	e := exporter.NewExcelExporter()
	data, err := e.Export(sampleResult())
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer f.Close()

	rows, err := f.GetRows("Report")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, []string{"id", "name", "department"}, rows[0])
	assert.Equal(t, []string{"1", "Asha", "Engineering"}, rows[1])
	assert.Equal(t, []string{"2", "Ben"}, rows[2])

	assert.Equal(t, domain.FormatExcel, e.Format())
	assert.Equal(t, "xlsx", e.FileExtension())
}
