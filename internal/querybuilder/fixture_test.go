package querybuilder_test

import "reporting-service/internal/domain"

// sampleTemplate mirrors the seeded "user_directory" demo report:
//
//	SELECT u.id, u.name, d.name AS department, u.status, u.created_at
//	FROM users u LEFT JOIN departments d ON u.department_id = d.id
func sampleTemplate() *domain.ReportTemplate {
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
			{TableAlias: "d", ColumnName: "name", Alias: "department", IsVisible: true, DisplayOrder: 3},
			{TableAlias: "u", ColumnName: "status", Alias: "status", IsVisible: true, DisplayOrder: 4},
			{TableAlias: "u", ColumnName: "created_at", Alias: "created_at", IsVisible: true, DisplayOrder: 5},
			{TableAlias: "u", ColumnName: "internal_note", Alias: "internal_note", IsVisible: false, DisplayOrder: 6},
		},
		Joins: []domain.ReportJoin{
			{JoinType: domain.JoinLeft, TableName: "departments", TableAlias: "d", LeftAlias: "u", LeftColumn: "department_id", RightAlias: "d", RightColumn: "id", JoinOrder: 1},
		},
		Filters: []domain.ReportFilter{
			{FieldName: "status", TableAlias: "u", ColumnName: "status", DataType: domain.DataTypeString, Operators: []domain.Operator{domain.OpEqual, domain.OpIn}},
			{FieldName: "created_at", TableAlias: "u", ColumnName: "created_at", DataType: domain.DataTypeDateTime, Operators: []domain.Operator{domain.OpBetween, domain.OpGreaterEq, domain.OpLessEq}},
			{FieldName: "department", TableAlias: "d", ColumnName: "name", DataType: domain.DataTypeString, Operators: []domain.Operator{domain.OpEqual, domain.OpContains}},
			{FieldName: "required_flag", TableAlias: "u", ColumnName: "id", DataType: domain.DataTypeInt, Operators: []domain.Operator{domain.OpEqual}, Required: true},
		},
		Sorts: []domain.ReportSort{
			{FieldName: "created_at", TableAlias: "u", ColumnName: "created_at", DefaultDir: domain.SortDesc, Priority: 1},
			{FieldName: "name", TableAlias: "u", ColumnName: "name", DefaultDir: domain.SortAsc, Priority: 2},
		},
		Export: domain.ReportExport{AllowCSV: true, AllowExcel: true, AllowJSON: true, MaxRows: 50000},
	}
}

// sampleTemplateNoRequiredFilter is sampleTemplate without the
// required_flag filter, for tests that don't want to supply it every time.
func sampleTemplateNoRequiredFilter() *domain.ReportTemplate {
	t := sampleTemplate()
	filters := make([]domain.ReportFilter, 0, len(t.Filters))
	for _, f := range t.Filters {
		if f.FieldName != "required_flag" {
			filters = append(filters, f)
		}
	}
	t.Filters = filters
	return t
}
