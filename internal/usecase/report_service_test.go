package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reporting-service/internal/domain"
	"reporting-service/internal/usecase"
)

// fakeRepo is an in-memory domain.ReportRepository for usecase tests.
type fakeRepo struct {
	templates map[int64]*domain.ReportTemplate
	execFunc  func(sql string, args []any, columns []string, countSQL string, countArgs []any) (*domain.ReportResult, error)
}

func (f *fakeRepo) ListTemplates(ctx context.Context) ([]domain.ReportTemplate, error) {
	var out []domain.ReportTemplate
	for _, t := range f.templates {
		out = append(out, *t)
	}
	return out, nil
}

func (f *fakeRepo) GetTemplate(ctx context.Context, reportID int64) (*domain.ReportTemplate, error) {
	t, ok := f.templates[reportID]
	if !ok {
		return nil, domain.NewNotFoundError("report %d not found", reportID)
	}
	return t, nil
}

func (f *fakeRepo) Execute(ctx context.Context, sql string, args []any, columns []string, countSQL string, countArgs []any) (*domain.ReportResult, error) {
	if f.execFunc != nil {
		return f.execFunc(sql, args, columns, countSQL, countArgs)
	}
	return &domain.ReportResult{Columns: columns, Rows: []map[string]any{}, TotalRows: 0}, nil
}

// fakeExporter and fakeRegistry let tests control export behavior without
// touching the real CSV/Excel/JSON implementations.
type fakeExporter struct {
	format domain.ExportFormat
	data   []byte
	err    error
}

func (e *fakeExporter) Format() domain.ExportFormat { return e.format }
func (e *fakeExporter) ContentType() string         { return "text/plain" }
func (e *fakeExporter) FileExtension() string       { return string(e.format) }
func (e *fakeExporter) Export(result *domain.ReportResult) ([]byte, error) {
	return e.data, e.err
}

type fakeRegistry struct {
	exporters map[domain.ExportFormat]domain.Exporter
}

func (r *fakeRegistry) Get(format domain.ExportFormat) (domain.Exporter, error) {
	e, ok := r.exporters[format]
	if !ok {
		return nil, assertErr{format}
	}
	return e, nil
}

type assertErr struct{ format domain.ExportFormat }

func (e assertErr) Error() string { return "unsupported format: " + string(e.format) }

func newSampleTemplate() *domain.ReportTemplate {
	return &domain.ReportTemplate{
		ID:          1,
		Name:        "user_directory",
		BaseTable:   "users",
		BaseAlias:   "u",
		Enabled:     true,
		MaxPageSize: 200,
		Columns: []domain.ReportColumn{
			{TableAlias: "u", ColumnName: "id", Alias: "id", IsVisible: true, DisplayOrder: 1},
			{TableAlias: "u", ColumnName: "name", Alias: "name", IsVisible: true, DisplayOrder: 2},
		},
		Filters: []domain.ReportFilter{
			{FieldName: "status", TableAlias: "u", ColumnName: "status", Operators: []domain.Operator{domain.OpEqual}},
		},
		Export: domain.ReportExport{AllowCSV: true, AllowExcel: false, AllowJSON: true, MaxRows: 100},
	}
}

func TestListReports_Passthrough(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	list, err := svc.ListReports(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "user_directory", list[0].Name)
}

func TestGetReport_Passthrough(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	template, err := svc.GetReport(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "user_directory", template.Name)

	_, err = svc.GetReport(context.Background(), 404)
	assert.Error(t, err)
}

func TestExport_ReportNotFound(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	_, _, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 99}, domain.FormatCSV)
	assert.Error(t, err)
}

func TestExport_JSONFormatNotAllowed(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	template := newSampleTemplate()
	template.Export.AllowJSON = false
	repo.templates[1] = template

	_, _, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 1}, domain.FormatJSON)
	assert.Error(t, err)
}

func TestExport_UnsupportedFormat(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	_, _, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 1}, "pdf")
	assert.Error(t, err)
}

func TestExport_ValidationErrorPropagates(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	req := &domain.ReportRequest{ReportID: 1, Filters: []domain.FilterCriterion{{Field: "unknown", Operator: domain.OpEqual, Value: "x"}}}
	_, _, err := svc.Export(context.Background(), req, domain.FormatCSV)
	assert.Error(t, err)
}

func TestExport_ExecutionErrorPropagates(t *testing.T) {
	repo := &fakeRepo{
		templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()},
		execFunc: func(sql string, args []any, columns []string, countSQL string, countArgs []any) (*domain.ReportResult, error) {
			return nil, domain.WrapInternal(assertErr{}, "db exploded")
		},
	}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	_, _, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 1}, domain.FormatCSV)
	assert.Error(t, err)
}

func TestExport_ExporterRenderErrorPropagates(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	registry := &fakeRegistry{exporters: map[domain.ExportFormat]domain.Exporter{
		domain.FormatCSV: &fakeExporter{format: domain.FormatCSV, err: assertErr{}},
	}}
	svc := usecase.NewReportService(repo, registry, nil)

	_, _, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 1}, domain.FormatCSV)
	assert.Error(t, err)
}

func TestGenerate_Success(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	result, err := svc.Generate(context.Background(), &domain.ReportRequest{ReportID: 1})
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "name"}, result.Columns)
}

func TestGenerate_ReportNotFound(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	_, err := svc.Generate(context.Background(), &domain.ReportRequest{ReportID: 99})
	assert.Error(t, err)
}

func TestGenerate_ValidationErrorPropagates(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	req := &domain.ReportRequest{
		ReportID: 1,
		Filters:  []domain.FilterCriterion{{Field: "unknown_field", Operator: domain.OpEqual, Value: "x"}},
	}
	_, err := svc.Generate(context.Background(), req)
	assert.Error(t, err)
}

func TestGenerate_ExecutionErrorPropagates(t *testing.T) {
	repo := &fakeRepo{
		templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()},
		execFunc: func(sql string, args []any, columns []string, countSQL string, countArgs []any) (*domain.ReportResult, error) {
			return nil, domain.WrapInternal(assertErr{}, "db exploded")
		},
	}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	_, err := svc.Generate(context.Background(), &domain.ReportRequest{ReportID: 1})
	assert.Error(t, err)
}

func TestExport_FormatNotAllowed(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	svc := usecase.NewReportService(repo, &fakeRegistry{}, nil)

	// Excel is disabled on this template.
	_, _, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 1}, domain.FormatExcel)
	require.Error(t, err)
	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.KindValidation, domErr.Kind)
}

func TestExport_Success(t *testing.T) {
	repo := &fakeRepo{templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()}}
	registry := &fakeRegistry{exporters: map[domain.ExportFormat]domain.Exporter{
		domain.FormatCSV: &fakeExporter{format: domain.FormatCSV, data: []byte("id,name\n1,Asha\n")},
	}}
	svc := usecase.NewReportService(repo, registry, nil)

	data, exp, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 1}, domain.FormatCSV)
	require.NoError(t, err)
	assert.Equal(t, "id,name\n1,Asha\n", string(data))
	assert.Equal(t, domain.FormatCSV, exp.Format())
}

func TestExport_MaxRowsExceedingMaxPageSizeStillWorks(t *testing.T) {
	// Regression: export must not be rejected by the /generate page-size
	// cap when MaxRows (a bulk-export bound) exceeds MaxPageSize.
	template := newSampleTemplate()
	template.MaxPageSize = 50
	template.Export.MaxRows = 50000

	var capturedArgs []any
	repo := &fakeRepo{
		templates: map[int64]*domain.ReportTemplate{1: template},
		execFunc: func(sql string, args []any, columns []string, countSQL string, countArgs []any) (*domain.ReportResult, error) {
			capturedArgs = args
			return &domain.ReportResult{Columns: columns}, nil
		},
	}
	registry := &fakeRegistry{exporters: map[domain.ExportFormat]domain.Exporter{
		domain.FormatCSV: &fakeExporter{format: domain.FormatCSV, data: []byte("ok")},
	}}
	svc := usecase.NewReportService(repo, registry, nil)

	_, _, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 1}, domain.FormatCSV)
	require.NoError(t, err)
	assert.Contains(t, capturedArgs, 50000)
}

func TestExport_LimitCappedToMaxRows(t *testing.T) {
	var capturedArgs []any
	repo := &fakeRepo{
		templates: map[int64]*domain.ReportTemplate{1: newSampleTemplate()},
		execFunc: func(sql string, args []any, columns []string, countSQL string, countArgs []any) (*domain.ReportResult, error) {
			capturedArgs = args
			return &domain.ReportResult{Columns: columns}, nil
		},
	}
	registry := &fakeRegistry{exporters: map[domain.ExportFormat]domain.Exporter{
		domain.FormatCSV: &fakeExporter{format: domain.FormatCSV, data: []byte("ok")},
	}}
	svc := usecase.NewReportService(repo, registry, nil)

	_, _, err := svc.Export(context.Background(), &domain.ReportRequest{ReportID: 1, Limit: 999999}, domain.FormatCSV)
	require.NoError(t, err)
	// LIMIT is the second-to-last bound arg; template MaxRows is 100.
	require.NotEmpty(t, capturedArgs)
	assert.Contains(t, capturedArgs, 100)
}
