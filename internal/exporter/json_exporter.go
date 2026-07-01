package exporter

import (
	"encoding/json"

	"reporting-service/internal/domain"
)

type JSONExporter struct{}

func NewJSONExporter() *JSONExporter { return &JSONExporter{} }

func (e *JSONExporter) Format() domain.ExportFormat { return domain.FormatJSON }
func (e *JSONExporter) ContentType() string         { return "application/json" }
func (e *JSONExporter) FileExtension() string       { return "json" }

func (e *JSONExporter) Export(result *domain.ReportResult) ([]byte, error) {
	return json.Marshal(result.Rows)
}
