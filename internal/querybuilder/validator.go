package querybuilder

import (
	"reporting-service/internal/domain"
)

const defaultPageSize = 50

// Validate checks a ReportRequest against the whitelists declared on
// template (filters, sorts) and rejects anything that doesn't match:
// unknown fields, disallowed operators, malformed values. This is the
// single gate that keeps user-supplied "field" values from ever reaching
// SQL as anything but a lookup key into trusted config.
func Validate(template *domain.ReportTemplate, req *domain.ReportRequest) error {
	filterByField := make(map[string]domain.ReportFilter, len(template.Filters))
	for _, f := range template.Filters {
		filterByField[f.FieldName] = f
	}

	seen := make(map[string]bool, len(req.Filters))
	for _, fc := range req.Filters {
		def, ok := filterByField[fc.Field]
		if !ok {
			return domain.NewValidationError("unknown filter field %q", fc.Field)
		}
		seen[fc.Field] = true

		if !isKnownOperator(fc.Operator) {
			return domain.NewValidationError("unsupported operator %q", fc.Operator)
		}
		if !operatorAllowed(def.Operators, fc.Operator) {
			return domain.NewValidationError("operator %q not allowed for field %q", fc.Operator, fc.Field)
		}
		if fc.Operator != domain.OpIsNull && fc.Operator != domain.OpIsNotNull && fc.Value == nil {
			return domain.NewValidationError("field %q requires a value", fc.Field)
		}
	}

	for _, f := range template.Filters {
		if f.Required && !seen[f.FieldName] {
			return domain.NewValidationError("missing required filter %q", f.FieldName)
		}
	}

	sortByField := make(map[string]bool, len(template.Sorts))
	for _, s := range template.Sorts {
		sortByField[s.FieldName] = true
	}
	for _, sc := range req.Sort {
		if !sortByField[sc.Field] {
			return domain.NewValidationError("unknown sort field %q", sc.Field)
		}
		if sc.Direction != domain.SortAsc && sc.Direction != domain.SortDesc {
			return domain.NewValidationError("invalid sort direction %q", sc.Direction)
		}
	}

	if req.Page < 0 {
		return domain.NewValidationError("page must be >= 0")
	}
	if req.Limit < 0 {
		return domain.NewValidationError("limit must be >= 0")
	}
	maxPage := template.MaxPageSize
	if maxPage > 0 && req.Limit > maxPage {
		return domain.NewValidationError("limit %d exceeds max page size %d", req.Limit, maxPage)
	}

	return nil
}

func operatorAllowed(allowed []domain.Operator, op domain.Operator) bool {
	for _, a := range allowed {
		if a == op {
			return true
		}
	}
	return false
}

// NormalizePagination fills in defaults for page/limit when the caller
// omitted them (zero values), independent of validation.
func NormalizePagination(page, limit, maxPageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if maxPageSize > 0 && limit > maxPageSize {
		limit = maxPageSize
	}
	return page, limit
}
