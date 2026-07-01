package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"reporting-service/internal/domain"
	"reporting-service/pkg/response"
)

const previewMaxLimit = 20

// ReportServicer is the subset of usecase.ReportService the HTTP layer
// depends on, so handlers can be tested against a fake.
type ReportServicer interface {
	ListReports(ctx context.Context) ([]domain.ReportTemplate, error)
	GetReport(ctx context.Context, reportID int64) (*domain.ReportTemplate, error)
	Generate(ctx context.Context, req *domain.ReportRequest) (*domain.ReportResult, error)
	Export(ctx context.Context, req *domain.ReportRequest, format domain.ExportFormat) ([]byte, domain.Exporter, error)
}

type ReportHandler struct {
	svc ReportServicer
}

func NewReportHandler(svc ReportServicer) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	templates, err := h.svc.ListReports(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	out := make([]templateSummaryDTO, len(templates))
	for i, t := range templates {
		out[i] = toTemplateSummaryDTO(t)
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *ReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	reportID, err := parseReportID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	template, err := h.svc.GetReport(r.Context(), reportID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toTemplateDetailDTO(template))
}

func (h *ReportHandler) Generate(w http.ResponseWriter, r *http.Request) {
	h.runGenerate(w, r, false)
}

func (h *ReportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	h.runGenerate(w, r, true)
}

func (h *ReportHandler) runGenerate(w http.ResponseWriter, r *http.Request, preview bool) {
	reportID, err := parseReportID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var body reportRequestDTO
	if err := decodeJSON(r, &body); err != nil {
		response.Error(w, domain.NewValidationError("invalid request body: %v", err))
		return
	}

	req := body.toDomain(reportID)
	if preview {
		if req.Limit <= 0 || req.Limit > previewMaxLimit {
			req.Limit = previewMaxLimit
		}
		req.Page = 1
	}

	result, err := h.svc.Generate(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toResultDTO(reportID, result))
}

func (h *ReportHandler) Export(w http.ResponseWriter, r *http.Request) {
	reportID, err := parseReportID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var body reportRequestDTO
	if err := decodeJSON(r, &body); err != nil {
		response.Error(w, domain.NewValidationError("invalid request body: %v", err))
		return
	}

	format := domain.ExportFormat(body.Format)
	if format == "" {
		format = domain.FormatCSV
	}

	req := body.toDomain(reportID)
	data, exp, err := h.svc.Export(r.Context(), req, format)
	if err != nil {
		response.Error(w, err)
		return
	}

	filename := fmt.Sprintf("report_%d.%s", reportID, exp.FileExtension())
	w.Header().Set("Content-Type", exp.ContentType())
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func parseReportID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, domain.NewValidationError("invalid report id %q", idStr)
	}
	return id, nil
}

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}
