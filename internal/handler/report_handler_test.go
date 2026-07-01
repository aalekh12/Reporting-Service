package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reporting-service/internal/domain"
	"reporting-service/internal/handler"
	"reporting-service/pkg/logger"
)

type fakeService struct {
	listResult []domain.ReportTemplate
	listErr    error

	getResult *domain.ReportTemplate
	getErr    error

	generateResult *domain.ReportResult
	generateErr    error
	lastRequest    *domain.ReportRequest

	exportData []byte
	exportExp  domain.Exporter
	exportErr  error
}

func (f *fakeService) ListReports(ctx context.Context) ([]domain.ReportTemplate, error) {
	return f.listResult, f.listErr
}

func (f *fakeService) GetReport(ctx context.Context, reportID int64) (*domain.ReportTemplate, error) {
	return f.getResult, f.getErr
}

func (f *fakeService) Generate(ctx context.Context, req *domain.ReportRequest) (*domain.ReportResult, error) {
	f.lastRequest = req
	return f.generateResult, f.generateErr
}

func (f *fakeService) Export(ctx context.Context, req *domain.ReportRequest, format domain.ExportFormat) ([]byte, domain.Exporter, error) {
	f.lastRequest = req
	return f.exportData, f.exportExp, f.exportErr
}

type stubExporter struct{ ext, contentType string }

func (s stubExporter) Format() domain.ExportFormat                 { return domain.FormatCSV }
func (s stubExporter) ContentType() string                         { return s.contentType }
func (s stubExporter) FileExtension() string                       { return s.ext }
func (s stubExporter) Export(*domain.ReportResult) ([]byte, error) { return nil, nil }

func newTestRouter(svc handler.ReportServicer) http.Handler {
	return handler.NewRouter(svc, logger.New(false))
}

func TestListReports(t *testing.T) {
	svc := &fakeService{listResult: []domain.ReportTemplate{{ID: 1, Name: "user_directory"}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/", nil)
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user_directory")
}

func TestListReports_Error(t *testing.T) {
	svc := &fakeService{listErr: domain.WrapInternal(nil, "db down")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/", nil)
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetReport_Success(t *testing.T) {
	svc := &fakeService{getResult: &domain.ReportTemplate{
		ID:   1,
		Name: "user_directory",
		Columns: []domain.ReportColumn{
			{Alias: "id", DataType: domain.DataTypeInt, IsVisible: true},
			{Alias: "hidden", DataType: domain.DataTypeString, IsVisible: false},
		},
		Filters: []domain.ReportFilter{
			{FieldName: "status", DataType: domain.DataTypeString, Operators: []domain.Operator{domain.OpEqual}, Required: true},
		},
		Sorts:  []domain.ReportSort{{FieldName: "created_at", DefaultDir: domain.SortDesc}},
		Export: domain.ReportExport{AllowCSV: true, AllowExcel: true, AllowJSON: true, MaxRows: 1000},
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/1", nil)
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	columns := out["columns"].([]any)
	assert.Len(t, columns, 1) // hidden column excluded
	filters := out["filters"].([]any)
	assert.Len(t, filters, 1)
}

func TestGetReport_NotFound(t *testing.T) {
	svc := &fakeService{getErr: domain.NewNotFoundError("report %d not found", 99)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/99", nil)
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetReport_InvalidID(t *testing.T) {
	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/not-a-number", nil)
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGenerate_Success(t *testing.T) {
	svc := &fakeService{generateResult: &domain.ReportResult{Columns: []string{"id"}, Rows: []map[string]any{{"id": 1}}}}
	body := `{"filters":[{"field":"status","operator":"=","value":"Active"}],"page":1,"limit":50}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/generate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, float64(1), out["report_id"])

	require.NotNil(t, svc.lastRequest)
	assert.Equal(t, domain.OpEqual, svc.lastRequest.Filters[0].Operator)
}

func TestGenerate_ValidationError(t *testing.T) {
	svc := &fakeService{generateErr: domain.NewValidationError("unknown filter field")}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/generate", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGenerate_MalformedBody(t *testing.T) {
	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/generate", bytes.NewBufferString(`{"page": "not-a-number"}`))
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPreview_CapsLimit(t *testing.T) {
	svc := &fakeService{generateResult: &domain.ReportResult{Columns: []string{"id"}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/preview", bytes.NewBufferString(`{"limit":5000}`))
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, svc.lastRequest)
	assert.Equal(t, 20, svc.lastRequest.Limit)
	assert.Equal(t, 1, svc.lastRequest.Page)
}

func TestExport_Success(t *testing.T) {
	svc := &fakeService{exportData: []byte("id,name\n1,Asha\n"), exportExp: stubExporter{ext: "csv", contentType: "text/csv"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/export", bytes.NewBufferString(`{"format":"csv"}`))
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "report_1.csv")
	assert.Equal(t, "id,name\n1,Asha\n", w.Body.String())
}

func TestExport_ServiceError(t *testing.T) {
	svc := &fakeService{exportErr: domain.NewValidationError("csv export is not enabled for this report")}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/export", bytes.NewBufferString(`{"format":"csv"}`))
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExport_MalformedBody(t *testing.T) {
	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/export", bytes.NewBufferString(`{"page": "nope"}`))
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExport_DefaultsToCSVFormat(t *testing.T) {
	svc := &fakeService{exportData: []byte("data"), exportExp: stubExporter{ext: "csv", contentType: "text/csv"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/1/export", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestHealth(t *testing.T) {
	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}
