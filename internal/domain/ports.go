package domain

import "context"

// ReportRepository is the persistence port for report configuration and
// report execution. Implemented by internal/repository/postgres.
type ReportRepository interface {
	// ListTemplates returns all enabled report templates (metadata only,
	// no columns/joins/filters/sorts/groups).
	ListTemplates(ctx context.Context) ([]ReportTemplate, error)

	// GetTemplate loads a single template with its full configuration
	// (columns, joins, filters, sorts, groups, export settings).
	GetTemplate(ctx context.Context, reportID int64) (*ReportTemplate, error)

	// Execute runs a parameterized query and returns rows preserving the
	// given column order, plus a total row count for pagination.
	Execute(ctx context.Context, sql string, args []any, columns []string, countSQL string, countArgs []any) (*ReportResult, error)
}

// Exporter renders a ReportResult into a byte stream of a given format.
type Exporter interface {
	Format() ExportFormat
	ContentType() string
	FileExtension() string
	Export(result *ReportResult) ([]byte, error)
}
