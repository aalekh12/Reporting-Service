package exporter

import (
	"fmt"

	"reporting-service/internal/domain"
)

// Registry looks up the Exporter for a requested format.
type Registry struct {
	exporters map[domain.ExportFormat]domain.Exporter
}

func NewRegistry(exporters ...domain.Exporter) *Registry {
	r := &Registry{exporters: make(map[domain.ExportFormat]domain.Exporter, len(exporters))}
	for _, e := range exporters {
		r.exporters[e.Format()] = e
	}
	return r
}

// DefaultRegistry wires the built-in JSON/CSV/XLSX exporters.
func DefaultRegistry() *Registry {
	return NewRegistry(NewJSONExporter(), NewCSVExporter(), NewExcelExporter())
}

func (r *Registry) Get(format domain.ExportFormat) (domain.Exporter, error) {
	e, ok := r.exporters[format]
	if !ok {
		return nil, fmt.Errorf("unsupported export format %q", format)
	}
	return e, nil
}
