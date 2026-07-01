package querybuilder_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"reporting-service/internal/domain"
	"reporting-service/internal/querybuilder"
)

func TestValidate_Success(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{
		ReportID: 1,
		Filters:  []domain.FilterCriterion{{Field: "status", Operator: domain.OpEqual, Value: "Active"}},
		Sort:     []domain.SortCriterion{{Field: "created_at", Direction: domain.SortDesc}},
		Page:     1,
		Limit:    50,
	}
	assert.NoError(t, querybuilder.Validate(template, req))
}

func TestValidate_UnknownFilterField(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Filters: []domain.FilterCriterion{{Field: "ssn", Operator: domain.OpEqual, Value: "x"}}}
	err := querybuilder.Validate(template, req)
	assert.Error(t, err)
	assertValidationKind(t, err)
}

func TestValidate_UnknownOperator(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Filters: []domain.FilterCriterion{{Field: "status", Operator: "DROP TABLE", Value: "x"}}}
	err := querybuilder.Validate(template, req)
	assert.Error(t, err)
}

func TestValidate_OperatorNotAllowedForField(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	// "status" only allows "=" and "in", not "between".
	req := &domain.ReportRequest{Filters: []domain.FilterCriterion{{Field: "status", Operator: domain.OpBetween, Value: []any{"a", "b"}}}}
	err := querybuilder.Validate(template, req)
	assert.Error(t, err)
}

func TestValidate_MissingValue(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Filters: []domain.FilterCriterion{{Field: "status", Operator: domain.OpEqual, Value: nil}}}
	assert.Error(t, querybuilder.Validate(template, req))
}

func TestValidate_IsNullDoesNotRequireValue(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	template.Filters[0].Operators = append(template.Filters[0].Operators, domain.OpIsNull)
	req := &domain.ReportRequest{Filters: []domain.FilterCriterion{{Field: "status", Operator: domain.OpIsNull, Value: nil}}}
	assert.NoError(t, querybuilder.Validate(template, req))
}

func TestValidate_MissingRequiredFilter(t *testing.T) {
	template := sampleTemplate() // includes required_flag
	req := &domain.ReportRequest{}
	err := querybuilder.Validate(template, req)
	assert.Error(t, err)
}

func TestValidate_RequiredFilterProvided(t *testing.T) {
	template := sampleTemplate()
	req := &domain.ReportRequest{Filters: []domain.FilterCriterion{{Field: "required_flag", Operator: domain.OpEqual, Value: 1}}}
	assert.NoError(t, querybuilder.Validate(template, req))
}

func TestValidate_UnknownSortField(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Sort: []domain.SortCriterion{{Field: "secret", Direction: domain.SortAsc}}}
	assert.Error(t, querybuilder.Validate(template, req))
}

func TestValidate_InvalidSortDirection(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Sort: []domain.SortCriterion{{Field: "name", Direction: "sideways"}}}
	assert.Error(t, querybuilder.Validate(template, req))
}

func TestValidate_NegativePageOrLimit(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	assert.Error(t, querybuilder.Validate(template, &domain.ReportRequest{Page: -1}))
	assert.Error(t, querybuilder.Validate(template, &domain.ReportRequest{Limit: -1}))
}

func TestValidate_LimitExceedsMaxPageSize(t *testing.T) {
	template := sampleTemplateNoRequiredFilter()
	req := &domain.ReportRequest{Limit: template.MaxPageSize + 1}
	assert.Error(t, querybuilder.Validate(template, req))
}

func TestNormalizePagination(t *testing.T) {
	page, limit := querybuilder.NormalizePagination(0, 0, 200)
	assert.Equal(t, 1, page)
	assert.Equal(t, 50, limit)

	page, limit = querybuilder.NormalizePagination(3, 500, 200)
	assert.Equal(t, 3, page)
	assert.Equal(t, 200, limit)

	page, limit = querybuilder.NormalizePagination(2, 10, 0)
	assert.Equal(t, 2, page)
	assert.Equal(t, 10, limit)
}

func assertValidationKind(t *testing.T, e error) {
	t.Helper()
	var domErr *domain.Error
	if !errors.As(e, &domErr) {
		t.Fatalf("expected *domain.Error, got %T", e)
	}
	assert.Equal(t, domain.KindValidation, domErr.Kind)
}
