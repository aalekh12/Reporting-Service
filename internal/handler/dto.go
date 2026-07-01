package handler

import "reporting-service/internal/domain"

type filterDTO struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

type sortDTO struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type reportRequestDTO struct {
	Filters []filterDTO `json:"filters"`
	Sort    []sortDTO   `json:"sort"`
	Page    int         `json:"page"`
	Limit   int         `json:"limit"`
	Format  string      `json:"format"`
}

func (d reportRequestDTO) toDomain(reportID int64) *domain.ReportRequest {
	req := &domain.ReportRequest{
		ReportID: reportID,
		Page:     d.Page,
		Limit:    d.Limit,
	}
	for _, f := range d.Filters {
		req.Filters = append(req.Filters, domain.FilterCriterion{
			Field:    f.Field,
			Operator: domain.Operator(f.Operator),
			Value:    f.Value,
		})
	}
	for _, s := range d.Sort {
		req.Sort = append(req.Sort, domain.SortCriterion{
			Field:     s.Field,
			Direction: domain.SortDirection(s.Direction),
		})
	}
	return req
}

type reportResultDTO struct {
	ReportID        int64            `json:"report_id"`
	Columns         []string         `json:"columns"`
	Rows            []map[string]any `json:"rows"`
	Page            int              `json:"page"`
	Limit           int              `json:"limit"`
	TotalRows       int              `json:"total_rows"`
	ExecutionTimeMs int64            `json:"execution_time_ms"`
}

func toResultDTO(reportID int64, r *domain.ReportResult) reportResultDTO {
	rows := r.Rows
	if rows == nil {
		rows = []map[string]any{}
	}
	return reportResultDTO{
		ReportID:        reportID,
		Columns:         r.Columns,
		Rows:            rows,
		Page:            r.Page,
		Limit:           r.Limit,
		TotalRows:       r.TotalRows,
		ExecutionTimeMs: r.ExecutionTimeMs,
	}
}

type templateSummaryDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func toTemplateSummaryDTO(t domain.ReportTemplate) templateSummaryDTO {
	return templateSummaryDTO{ID: t.ID, Name: t.Name, Description: t.Description}
}

type templateDetailDTO struct {
	ID          int64            `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Columns     []columnDTO      `json:"columns"`
	Filters     []filterFieldDTO `json:"filters"`
	Sorts       []sortFieldDTO   `json:"sorts"`
	Export      exportDTO        `json:"export"`
}

type columnDTO struct {
	Alias    string `json:"alias"`
	DataType string `json:"data_type"`
}

type filterFieldDTO struct {
	Field     string   `json:"field"`
	DataType  string   `json:"data_type"`
	Operators []string `json:"operators"`
	Required  bool     `json:"required"`
}

type sortFieldDTO struct {
	Field      string `json:"field"`
	DefaultDir string `json:"default_dir"`
}

type exportDTO struct {
	AllowCSV   bool `json:"allow_csv"`
	AllowExcel bool `json:"allow_excel"`
	AllowJSON  bool `json:"allow_json"`
	MaxRows    int  `json:"max_rows"`
}

func toTemplateDetailDTO(t *domain.ReportTemplate) templateDetailDTO {
	dto := templateDetailDTO{ID: t.ID, Name: t.Name, Description: t.Description}
	for _, c := range t.Columns {
		if c.IsVisible {
			dto.Columns = append(dto.Columns, columnDTO{Alias: c.Alias, DataType: string(c.DataType)})
		}
	}
	for _, f := range t.Filters {
		ops := make([]string, len(f.Operators))
		for i, o := range f.Operators {
			ops[i] = string(o)
		}
		dto.Filters = append(dto.Filters, filterFieldDTO{Field: f.FieldName, DataType: string(f.DataType), Operators: ops, Required: f.Required})
	}
	for _, s := range t.Sorts {
		dto.Sorts = append(dto.Sorts, sortFieldDTO{Field: s.FieldName, DefaultDir: string(s.DefaultDir)})
	}
	dto.Export = exportDTO{
		AllowCSV:   t.Export.AllowCSV,
		AllowExcel: t.Export.AllowExcel,
		AllowJSON:  t.Export.AllowJSON,
		MaxRows:    t.Export.MaxRows,
	}
	return dto
}
