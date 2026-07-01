package querybuilder_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reporting-service/internal/domain"
	"reporting-service/internal/querybuilder"
)

func TestBuild_BasicFilterAndDefaultSort(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{
		Filters: []domain.FilterCriterion{
			{Field: "status", Operator: domain.OpEqual, Value: "Active"},
			{Field: "created_at", Operator: domain.OpBetween, Value: []any{"2026-01-01", "2026-06-30"}},
		},
	}

	built, err := querybuilder.Build(template, req)
	require.NoError(t, err)

	assert.Equal(t,
		`SELECT "u"."id" AS "id", "u"."name" AS "name", "d"."name" AS "department", "u"."status" AS "status", "u"."created_at" AS "created_at"`+
			` FROM "users" AS "u" LEFT JOIN "departments" AS "d" ON "u"."department_id" = "d"."id"`+
			` WHERE "u"."status" = $1 AND "u"."created_at" BETWEEN $2 AND $3`+
			` ORDER BY "u"."created_at" DESC, "u"."name" ASC LIMIT $4 OFFSET $5`,
		built.SQL,
	)
	assert.Equal(t, []any{"Active", "2026-01-01", "2026-06-30", 50, 0}, built.Args)
	assert.Equal(t, []string{"id", "name", "department", "status", "created_at"}, built.Columns)
	assert.Equal(t, 1, built.Page)
	assert.Equal(t, 50, built.Limit)
}

func TestBuild_InvisibleColumnExcluded(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	built, err := querybuilder.Build(template, &domain.ReportRequest{})
	require.NoError(t, err)
	assert.NotContains(t, built.Columns, "internal_note")
}

func TestBuild_RequestSortOverridesDefault(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Sort: []domain.SortCriterion{{Field: "name", Direction: domain.SortAsc}}}

	built, err := querybuilder.Build(template, req)
	require.NoError(t, err)
	assert.Contains(t, built.SQL, `ORDER BY "u"."name" ASC`)
	assert.NotContains(t, built.SQL, "created_at\" DESC")
}

func TestBuild_Pagination(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Page: 3, Limit: 10}

	built, err := querybuilder.Build(template, req)
	require.NoError(t, err)
	assert.Contains(t, built.SQL, "LIMIT $1 OFFSET $2")
	assert.Equal(t, []any{10, 20}, built.Args)
	assert.Equal(t, 3, built.Page)
	assert.Equal(t, 10, built.Limit)
}

func TestBuild_LimitClampedToMaxPageSize(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	template.MaxPageSize = 5
	req := &domain.ReportRequest{Limit: 1000}

	built, err := querybuilder.Build(template, req)
	require.NoError(t, err)
	assert.Equal(t, 5, built.Limit)
}

func TestBuild_InClauseExpandsPlaceholders(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{
		Filters: []domain.FilterCriterion{{Field: "status", Operator: domain.OpIn, Value: []any{"Active", "Pending", "Onboarding"}}},
	}

	built, err := querybuilder.Build(template, req)
	require.NoError(t, err)
	assert.Contains(t, built.SQL, `"u"."status" IN ($1, $2, $3)`)
	assert.Equal(t, []any{"Active", "Pending", "Onboarding", 50, 0}, built.Args)
}

func TestBuild_ContainsUsesILIKEWithWildcards(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{
		Filters: []domain.FilterCriterion{{Field: "department", Operator: domain.OpContains, Value: "eng"}},
	}

	built, err := querybuilder.Build(template, req)
	require.NoError(t, err)
	assert.Contains(t, built.SQL, `"d"."name" ILIKE $1`)
	assert.Equal(t, "%eng%", built.Args[0])
}

func TestBuild_UnknownFilterFieldIsRejectedEvenIfValidationSkipped(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Filters: []domain.FilterCriterion{{Field: "not_whitelisted", Operator: domain.OpEqual, Value: "x"}}}

	_, err := querybuilder.Build(template, req)
	assert.Error(t, err)
}

func TestBuild_NoVisibleColumnsErrors(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	for i := range template.Columns {
		template.Columns[i].IsVisible = false
	}
	_, err := querybuilder.Build(template, &domain.ReportRequest{})
	assert.Error(t, err)
}

func TestBuild_MaliciousIdentifierInConfigIsRejected(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	template.Columns[0].ColumnName = `id"; DROP TABLE users; --`

	_, err := querybuilder.Build(template, &domain.ReportRequest{})
	assert.Error(t, err)
}

func TestBuild_CountSQLWithoutGroupBy(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Filters: []domain.FilterCriterion{{Field: "status", Operator: domain.OpEqual, Value: "Active"}}}

	built, err := querybuilder.Build(template, req)
	require.NoError(t, err)
	assert.Equal(t,
		`SELECT COUNT(*) FROM "users" AS "u" LEFT JOIN "departments" AS "d" ON "u"."department_id" = "d"."id" WHERE "u"."status" = $1`,
		built.CountSQL,
	)
	assert.Equal(t, []any{"Active"}, built.CountArgs)
}

func TestBuild_CountSQLWithGroupBy(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	template.Groups = []domain.ReportGroup{{TableAlias: "d", ColumnName: "name", DisplayOrder: 1}}

	built, err := querybuilder.Build(template, &domain.ReportRequest{})
	require.NoError(t, err)
	assert.Contains(t, built.SQL, `GROUP BY "d"."name"`)
	assert.Equal(t,
		`SELECT COUNT(*) FROM (SELECT 1 FROM "users" AS "u" LEFT JOIN "departments" AS "d" ON "u"."department_id" = "d"."id" GROUP BY "d"."name") AS sub`,
		built.CountSQL,
	)
}
