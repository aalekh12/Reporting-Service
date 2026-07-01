// Package usecase orchestrates: load report config -> validate request ->
// build SQL -> execute -> format/export. It depends only on domain
// interfaces (ports), never on pgx/chi/excelize directly, so it can be
// unit tested with in-memory fakes.
package usecase

import (
	"context"
	"log/slog"

	"reporting-service/internal/domain"
	"reporting-service/internal/querybuilder"
)

// ExporterRegistry resolves a domain.Exporter for a requested format.
type ExporterRegistry interface {
	Get(format domain.ExportFormat) (domain.Exporter, error)
}

type ReportService struct {
	repo      domain.ReportRepository
	exporters ExporterRegistry
	log       *slog.Logger
}

func NewReportService(repo domain.ReportRepository, exporters ExporterRegistry, log *slog.Logger) *ReportService {
	if log == nil {
		log = slog.Default()
	}
	return &ReportService{repo: repo, exporters: exporters, log: log}
}

func (s *ReportService) ListReports(ctx context.Context) ([]domain.ReportTemplate, error) {
	return s.repo.ListTemplates(ctx)
}

func (s *ReportService) GetReport(ctx context.Context, reportID int64) (*domain.ReportTemplate, error) {
	return s.repo.GetTemplate(ctx, reportID)
}

// Generate loads the report template, validates the request against its
// whitelists, builds the parameterized query, executes it, and returns the
// result. Used by both /generate and /preview (the handler caps Limit for
// preview before calling this).
func (s *ReportService) Generate(ctx context.Context, req *domain.ReportRequest) (*domain.ReportResult, error) {
	template, err := s.repo.GetTemplate(ctx, req.ReportID)
	if err != nil {
		return nil, err
	}

	if err := querybuilder.Validate(template, req); err != nil {
		return nil, err
	}

	built, err := querybuilder.Build(template, req)
	if err != nil {
		return nil, domain.WrapInternal(err, "build report query")
	}

	result, err := s.repo.Execute(ctx, built.SQL, built.Args, built.Columns, built.CountSQL, built.CountArgs)
	if err != nil {
		s.log.Error("report execution failed", "report_id", req.ReportID, "sql", built.SQL, "error", err)
		return nil, err
	}
	result.Page = built.Page
	result.Limit = built.Limit

	s.log.Info("report executed",
		"report_id", req.ReportID,
		"sql", built.SQL,
		"row_count", len(result.Rows),
		"total_rows", result.TotalRows,
		"execution_time_ms", result.ExecutionTimeMs,
	)

	return result, nil
}

// Export runs the report (capped to the template's max export row count)
// and renders it via the exporter registered for format.
func (s *ReportService) Export(ctx context.Context, req *domain.ReportRequest, format domain.ExportFormat) ([]byte, domain.Exporter, error) {
	template, err := s.repo.GetTemplate(ctx, req.ReportID)
	if err != nil {
		return nil, nil, err
	}

	if err := checkFormatAllowed(template.Export, format); err != nil {
		return nil, nil, err
	}

	exportReq := *req
	exportReq.Page = 1
	if exportReq.Limit <= 0 || exportReq.Limit > template.Export.MaxRows {
		exportReq.Limit = template.Export.MaxRows
	}

	// Export is a bulk operation bounded by MaxRows, not by the page-size
	// cap that applies to paginated /generate requests. Validate/Build
	// against a template whose effective page-size ceiling reflects that.
	exportTemplate := *template
	if exportTemplate.MaxPageSize < template.Export.MaxRows {
		exportTemplate.MaxPageSize = template.Export.MaxRows
	}

	if err := querybuilder.Validate(&exportTemplate, &exportReq); err != nil {
		return nil, nil, err
	}

	built, err := querybuilder.Build(&exportTemplate, &exportReq)
	if err != nil {
		return nil, nil, domain.WrapInternal(err, "build report query")
	}

	result, err := s.repo.Execute(ctx, built.SQL, built.Args, built.Columns, built.CountSQL, built.CountArgs)
	if err != nil {
		s.log.Error("report export failed", "report_id", req.ReportID, "sql", built.SQL, "error", err)
		return nil, nil, err
	}

	exp, err := s.exporters.Get(format)
	if err != nil {
		return nil, nil, domain.NewValidationError("%s", err.Error())
	}

	data, err := exp.Export(result)
	if err != nil {
		return nil, nil, domain.WrapInternal(err, "render export")
	}

	s.log.Info("report exported",
		"report_id", req.ReportID,
		"format", format,
		"row_count", len(result.Rows),
		"execution_time_ms", result.ExecutionTimeMs,
	)

	return data, exp, nil
}

func checkFormatAllowed(export domain.ReportExport, format domain.ExportFormat) error {
	switch format {
	case domain.FormatCSV:
		if !export.AllowCSV {
			return domain.NewValidationError("csv export is not enabled for this report")
		}
	case domain.FormatExcel:
		if !export.AllowExcel {
			return domain.NewValidationError("excel export is not enabled for this report")
		}
	case domain.FormatJSON:
		if !export.AllowJSON {
			return domain.NewValidationError("json export is not enabled for this report")
		}
	default:
		return domain.NewValidationError("unsupported export format %q", format)
	}
	return nil
}
