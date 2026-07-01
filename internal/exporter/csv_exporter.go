package exporter

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"reporting-service/internal/domain"
)

type CSVExporter struct{}

func NewCSVExporter() *CSVExporter { return &CSVExporter{} }

func (e *CSVExporter) Format() domain.ExportFormat { return domain.FormatCSV }
func (e *CSVExporter) ContentType() string         { return "text/csv" }
func (e *CSVExporter) FileExtension() string       { return "csv" }

func (e *CSVExporter) Export(result *domain.ReportResult) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write(result.Columns); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}
	for _, row := range result.Rows {
		record := make([]string, len(result.Columns))
		for i, col := range result.Columns {
			record[i] = formatValue(row[col])
		}
		if err := w.Write(record); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}
	return buf.Bytes(), nil
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
