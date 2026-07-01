// Package postgres implements domain.ReportRepository against a real
// PostgreSQL database via pgx.
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"reporting-service/internal/domain"
)

// DB is the minimal pool surface this package needs, satisfied by
// *pgxpool.Pool in production and by pgxmock in tests.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ReportRepository struct {
	db DB
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{db: pool}
}

// NewReportRepositoryWithDB allows injecting a mock DB (e.g. pgxmock) for tests.
func NewReportRepositoryWithDB(db DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) ListTemplates(ctx context.Context) ([]domain.ReportTemplate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, description, base_table, base_alias, enabled, max_page_size
		FROM report_templates
		WHERE enabled = TRUE
		ORDER BY name`)
	if err != nil {
		return nil, domain.WrapInternal(err, "list report templates")
	}
	defer rows.Close()

	var templates []domain.ReportTemplate
	for rows.Next() {
		var t domain.ReportTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.BaseTable, &t.BaseAlias, &t.Enabled, &t.MaxPageSize); err != nil {
			return nil, domain.WrapInternal(err, "scan report template")
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.WrapInternal(err, "iterate report templates")
	}
	return templates, nil
}

func (r *ReportRepository) GetTemplate(ctx context.Context, reportID int64) (*domain.ReportTemplate, error) {
	var t domain.ReportTemplate
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, base_table, base_alias, enabled, max_page_size
		FROM report_templates
		WHERE id = $1`, reportID,
	).Scan(&t.ID, &t.Name, &t.Description, &t.BaseTable, &t.BaseAlias, &t.Enabled, &t.MaxPageSize)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.NewNotFoundError("report %d not found", reportID)
		}
		return nil, domain.WrapInternal(err, "get report template")
	}
	if !t.Enabled {
		return nil, domain.NewNotFoundError("report %d not found", reportID)
	}

	var err2 error
	if t.Columns, err2 = r.loadColumns(ctx, reportID); err2 != nil {
		return nil, err2
	}
	if t.Joins, err2 = r.loadJoins(ctx, reportID); err2 != nil {
		return nil, err2
	}
	if t.Filters, err2 = r.loadFilters(ctx, reportID); err2 != nil {
		return nil, err2
	}
	if t.Sorts, err2 = r.loadSorts(ctx, reportID); err2 != nil {
		return nil, err2
	}
	if t.Groups, err2 = r.loadGroups(ctx, reportID); err2 != nil {
		return nil, err2
	}
	export, err2 := r.loadExport(ctx, reportID)
	if err2 != nil {
		return nil, err2
	}
	t.Export = export

	return &t, nil
}

func (r *ReportRepository) loadColumns(ctx context.Context, reportID int64) ([]domain.ReportColumn, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, report_id, table_alias, column_name, alias, COALESCE(expression, ''), data_type, is_visible, display_order
		FROM report_columns WHERE report_id = $1 ORDER BY display_order`, reportID)
	if err != nil {
		return nil, domain.WrapInternal(err, "load report columns")
	}
	defer rows.Close()

	var out []domain.ReportColumn
	for rows.Next() {
		var c domain.ReportColumn
		var dataType string
		if err := rows.Scan(&c.ID, &c.ReportID, &c.TableAlias, &c.ColumnName, &c.Alias, &c.Expression, &dataType, &c.IsVisible, &c.DisplayOrder); err != nil {
			return nil, domain.WrapInternal(err, "scan report column")
		}
		c.DataType = domain.DataType(dataType)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ReportRepository) loadJoins(ctx context.Context, reportID int64) ([]domain.ReportJoin, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, report_id, join_type, table_name, table_alias, left_alias, left_column, right_alias, right_column, join_order
		FROM report_joins WHERE report_id = $1 ORDER BY join_order`, reportID)
	if err != nil {
		return nil, domain.WrapInternal(err, "load report joins")
	}
	defer rows.Close()

	var out []domain.ReportJoin
	for rows.Next() {
		var j domain.ReportJoin
		var joinType string
		if err := rows.Scan(&j.ID, &j.ReportID, &joinType, &j.TableName, &j.TableAlias, &j.LeftAlias, &j.LeftColumn, &j.RightAlias, &j.RightColumn, &j.JoinOrder); err != nil {
			return nil, domain.WrapInternal(err, "scan report join")
		}
		j.JoinType = domain.JoinType(joinType)
		out = append(out, j)
	}
	return out, rows.Err()
}

func (r *ReportRepository) loadFilters(ctx context.Context, reportID int64) ([]domain.ReportFilter, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, report_id, field_name, table_alias, column_name, data_type, operators, required
		FROM report_filters WHERE report_id = $1`, reportID)
	if err != nil {
		return nil, domain.WrapInternal(err, "load report filters")
	}
	defer rows.Close()

	var out []domain.ReportFilter
	for rows.Next() {
		var f domain.ReportFilter
		var dataType string
		var ops []string
		if err := rows.Scan(&f.ID, &f.ReportID, &f.FieldName, &f.TableAlias, &f.ColumnName, &dataType, &ops, &f.Required); err != nil {
			return nil, domain.WrapInternal(err, "scan report filter")
		}
		f.DataType = domain.DataType(dataType)
		f.Operators = make([]domain.Operator, len(ops))
		for i, o := range ops {
			f.Operators[i] = domain.Operator(o)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *ReportRepository) loadSorts(ctx context.Context, reportID int64) ([]domain.ReportSort, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, report_id, field_name, table_alias, column_name, default_dir, priority
		FROM report_sorts WHERE report_id = $1 ORDER BY priority`, reportID)
	if err != nil {
		return nil, domain.WrapInternal(err, "load report sorts")
	}
	defer rows.Close()

	var out []domain.ReportSort
	for rows.Next() {
		var s domain.ReportSort
		var dir string
		if err := rows.Scan(&s.ID, &s.ReportID, &s.FieldName, &s.TableAlias, &s.ColumnName, &dir, &s.Priority); err != nil {
			return nil, domain.WrapInternal(err, "scan report sort")
		}
		s.DefaultDir = domain.SortDirection(dir)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ReportRepository) loadGroups(ctx context.Context, reportID int64) ([]domain.ReportGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, report_id, table_alias, column_name, display_order
		FROM report_groups WHERE report_id = $1 ORDER BY display_order`, reportID)
	if err != nil {
		return nil, domain.WrapInternal(err, "load report groups")
	}
	defer rows.Close()

	var out []domain.ReportGroup
	for rows.Next() {
		var g domain.ReportGroup
		if err := rows.Scan(&g.ID, &g.ReportID, &g.TableAlias, &g.ColumnName, &g.DisplayOrder); err != nil {
			return nil, domain.WrapInternal(err, "scan report group")
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *ReportRepository) loadExport(ctx context.Context, reportID int64) (domain.ReportExport, error) {
	var e domain.ReportExport
	err := r.db.QueryRow(ctx, `
		SELECT id, report_id, allow_csv, allow_excel, allow_json, max_rows
		FROM report_exports WHERE report_id = $1`, reportID,
	).Scan(&e.ID, &e.ReportID, &e.AllowCSV, &e.AllowExcel, &e.AllowJSON, &e.MaxRows)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No explicit export config: fall back to sane defaults.
			return domain.ReportExport{ReportID: reportID, AllowCSV: true, AllowExcel: true, AllowJSON: true, MaxRows: 10000}, nil
		}
		return e, domain.WrapInternal(err, "load report export settings")
	}
	return e, nil
}

// Execute runs sql/args to fetch a page of rows (scanned into ordered maps
// keyed by columns) and countSQL/countArgs to compute the total row count.
func (r *ReportRepository) Execute(ctx context.Context, sql string, args []any, columns []string, countSQL string, countArgs []any) (*domain.ReportResult, error) {
	start := time.Now()

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, domain.WrapInternal(err, "execute report query")
	}
	defer rows.Close()

	var result []map[string]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, domain.WrapInternal(err, "scan report row")
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			if i < len(values) {
				row[col] = values[i]
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.WrapInternal(err, "iterate report rows")
	}

	var total int64
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, domain.WrapInternal(err, "count report rows")
	}

	return &domain.ReportResult{
		Columns:         columns,
		Rows:            result,
		TotalRows:       int(total),
		ExecutionTimeMs: time.Since(start).Milliseconds(),
		GeneratedSQL:    sql,
	}, nil
}
