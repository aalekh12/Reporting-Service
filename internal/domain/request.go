package domain

// FilterCriterion is one runtime filter supplied by a caller. Field must
// match a ReportFilter.FieldName whitelisted for the target report.
type FilterCriterion struct {
	Field    string
	Operator Operator
	Value    any
}

// SortCriterion is one runtime sort supplied by a caller. Field must match
// a ReportSort.FieldName whitelisted for the target report.
type SortCriterion struct {
	Field     string
	Direction SortDirection
}

// ReportRequest is the transport-agnostic runtime request for executing a
// report: filters/sort come from the caller, everything else (columns,
// joins, groups) comes from the ReportTemplate.
type ReportRequest struct {
	ReportID int64
	Filters  []FilterCriterion
	Sort     []SortCriterion
	Page     int
	Limit    int
}

// ReportResult is the outcome of executing a report: ordered column aliases
// plus rows keyed by those same aliases, so exporters can render columns in
// the configured order without re-deriving it from the map.
type ReportResult struct {
	Columns         []string
	Rows            []map[string]any
	Page            int
	Limit           int
	TotalRows       int
	ExecutionTimeMs int64
	GeneratedSQL    string
}
