package postgres_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reporting-service/internal/domain"
	"reporting-service/internal/repository/postgres"
)

func newMockRepo(t *testing.T) (*postgres.ReportRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return postgres.NewReportRepositoryWithDB(mock), mock
}

func TestListTemplates(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, base_table, base_alias, enabled, max_page_size
		FROM report_templates
		WHERE enabled = TRUE
		ORDER BY name`)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "base_table", "base_alias", "enabled", "max_page_size"}).
			AddRow(int64(1), "user_directory", "desc", "users", "u", true, 200))

	templates, err := repo.ListTemplates(context.Background())
	require.NoError(t, err)
	require.Len(t, templates, 1)
	assert.Equal(t, "user_directory", templates[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTemplate_NotFound(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, base_table, base_alias, enabled, max_page_size
		FROM report_templates
		WHERE id = $1`)).
		WithArgs(int64(99)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "base_table", "base_alias", "enabled", "max_page_size"}))

	_, err := repo.GetTemplate(context.Background(), 99)
	require.Error(t, err)

	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.KindNotFound, domErr.Kind)
}

func TestGetTemplate_Disabled(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, base_table, base_alias, enabled, max_page_size
		FROM report_templates
		WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "base_table", "base_alias", "enabled", "max_page_size"}).
			AddRow(int64(1), "disabled_report", "", "users", "u", false, 200))

	_, err := repo.GetTemplate(context.Background(), 1)
	require.Error(t, err)
	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.KindNotFound, domErr.Kind)
}

func TestGetTemplate_FullyLoaded(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "base_table", "base_alias", "enabled", "max_page_size"}).
			AddRow(int64(1), "user_directory", "desc", "users", "u", true, 200))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_columns`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "table_alias", "column_name", "alias", "expression", "data_type", "is_visible", "display_order"}).
			AddRow(int64(1), int64(1), "u", "id", "id", "", "int", true, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_joins`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "join_type", "table_name", "table_alias", "left_alias", "left_column", "right_alias", "right_column", "join_order"}).
			AddRow(int64(1), int64(1), "LEFT", "departments", "d", "u", "department_id", "d", "id", 1))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_filters`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "field_name", "table_alias", "column_name", "data_type", "operators", "required"}).
			AddRow(int64(1), int64(1), "status", "u", "status", "string", []string{"="}, false))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_sorts`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "field_name", "table_alias", "column_name", "default_dir", "priority"}).
			AddRow(int64(1), int64(1), "created_at", "u", "created_at", "desc", 1))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_groups`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "table_alias", "column_name", "display_order"}).
			AddRow(int64(1), int64(1), "d", "name", 1))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_exports`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "allow_csv", "allow_excel", "allow_json", "max_rows"}).
			AddRow(int64(1), int64(1), true, true, true, 50000))

	template, err := repo.GetTemplate(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, template.Columns, 1)
	assert.Len(t, template.Joins, 1)
	assert.Len(t, template.Filters, 1)
	assert.Len(t, template.Sorts, 1)
	assert.Len(t, template.Groups, 1)
	assert.Equal(t, 50000, template.Export.MaxRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTemplate_ColumnsQueryError(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "base_table", "base_alias", "enabled", "max_page_size"}).
			AddRow(int64(1), "user_directory", "desc", "users", "u", true, 200))

	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_columns`)).
		WithArgs(int64(1)).
		WillReturnError(assertErr{})

	_, err := repo.GetTemplate(context.Background(), 1)
	assert.Error(t, err)
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }

func TestGetTemplate_MissingExportFallsBackToDefaults(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(`WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "base_table", "base_alias", "enabled", "max_page_size"}).
			AddRow(int64(1), "user_directory", "desc", "users", "u", true, 200))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_columns`)).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "table_alias", "column_name", "alias", "expression", "data_type", "is_visible", "display_order"}))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_joins`)).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "join_type", "table_name", "table_alias", "left_alias", "left_column", "right_alias", "right_column", "join_order"}))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_filters`)).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "field_name", "table_alias", "column_name", "data_type", "operators", "required"}))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_sorts`)).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "field_name", "table_alias", "column_name", "default_dir", "priority"}))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_groups`)).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "table_alias", "column_name", "display_order"}))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM report_exports`)).WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "report_id", "allow_csv", "allow_excel", "allow_json", "max_rows"}))

	template, err := repo.GetTemplate(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, template.Export.AllowCSV)
	assert.Equal(t, 10000, template.Export.MaxRows)
}

func TestExecute(t *testing.T) {
	repo, mock := newMockRepo(t)

	sql := `SELECT "u"."id" AS "id" FROM "users" AS "u" LIMIT $1 OFFSET $2`
	countSQL := `SELECT COUNT(*) FROM "users" AS "u"`

	mock.ExpectQuery(regexp.QuoteMeta(sql)).
		WithArgs(50, 0).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2)))

	mock.ExpectQuery(regexp.QuoteMeta(countSQL)).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(2)))

	result, err := repo.Execute(context.Background(), sql, []any{50, 0}, []string{"id"}, countSQL, nil)
	require.NoError(t, err)
	require.Len(t, result.Rows, 2)
	assert.Equal(t, int64(1), result.Rows[0]["id"])
	assert.Equal(t, 2, result.TotalRows)
	require.NoError(t, mock.ExpectationsWereMet())
}
