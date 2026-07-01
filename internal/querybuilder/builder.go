package querybuilder

import (
	"fmt"
	"sort"
	"strings"

	"reporting-service/internal/domain"
)

// Built is the output of Build: a ready-to-execute parameterized query for
// the page of data plus a matching query to compute the total row count.
type Built struct {
	SQL       string
	Args      []any
	CountSQL  string
	CountArgs []any
	Columns   []string // output column aliases, in display order
	Page      int
	Limit     int
}

func quoteIdent(name string) string {
	return `"` + name + `"`
}

// Build assembles a parameterized SELECT (+ matching COUNT) for template
// given a validated runtime request. Callers must run Validate first —
// Build does not re-check whitelists, only renders what it's given.
func Build(template *domain.ReportTemplate, req *domain.ReportRequest) (*Built, error) {
	if err := validIdentifier(template.BaseTable); err != nil {
		return nil, err
	}
	if err := validIdentifier(template.BaseAlias); err != nil {
		return nil, err
	}

	selectClause, columns, err := buildSelect(template.Columns)
	if err != nil {
		return nil, err
	}

	fromClause := fmt.Sprintf("%s AS %s", quoteIdent(template.BaseTable), quoteIdent(template.BaseAlias))

	joinClause, err := buildJoins(template.Joins)
	if err != nil {
		return nil, err
	}

	groupByClause, err := buildGroupBy(template.Groups)
	if err != nil {
		return nil, err
	}

	orderByClause, err := buildOrderBy(template.Sorts, req.Sort)
	if err != nil {
		return nil, err
	}

	page, limit := NormalizePagination(req.Page, req.Limit, template.MaxPageSize)
	offset := (page - 1) * limit

	binder := &paramBinder{}
	whereClause, err := buildWhere(template.Filters, req.Filters, binder)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString("SELECT ")
	b.WriteString(selectClause)
	b.WriteString(" FROM ")
	b.WriteString(fromClause)
	if joinClause != "" {
		b.WriteString(" ")
		b.WriteString(joinClause)
	}
	if whereClause != "" {
		b.WriteString(" WHERE ")
		b.WriteString(whereClause)
	}
	if groupByClause != "" {
		b.WriteString(" GROUP BY ")
		b.WriteString(groupByClause)
	}
	if orderByClause != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(orderByClause)
	}
	b.WriteString(fmt.Sprintf(" LIMIT %s OFFSET %s", binder.bind(limit), binder.bind(offset)))

	countBinder := &paramBinder{}
	countWhere, err := buildWhere(template.Filters, req.Filters, countBinder)
	if err != nil {
		return nil, err
	}
	countSQL := buildCountSQL(fromClause, joinClause, countWhere, groupByClause)

	return &Built{
		SQL:       b.String(),
		Args:      binder.args,
		CountSQL:  countSQL,
		CountArgs: countBinder.args,
		Columns:   columns,
		Page:      page,
		Limit:     limit,
	}, nil
}

func buildSelect(cols []domain.ReportColumn) (string, []string, error) {
	visible := make([]domain.ReportColumn, 0, len(cols))
	for _, c := range cols {
		if c.IsVisible {
			visible = append(visible, c)
		}
	}
	sort.SliceStable(visible, func(i, j int) bool { return visible[i].DisplayOrder < visible[j].DisplayOrder })

	if len(visible) == 0 {
		return "", nil, fmt.Errorf("report has no visible columns")
	}

	parts := make([]string, 0, len(visible))
	aliases := make([]string, 0, len(visible))
	for _, c := range visible {
		if err := validIdentifier(c.Alias); err != nil {
			return "", nil, err
		}
		var expr string
		if c.Expression != "" {
			// Expressions are operator-authored config, not user input;
			// still not string-built from request data.
			expr = c.Expression
		} else {
			q, err := qualify(c.TableAlias, c.ColumnName)
			if err != nil {
				return "", nil, err
			}
			expr = quoteQualified(q)
		}
		parts = append(parts, fmt.Sprintf("%s AS %s", expr, quoteIdent(c.Alias)))
		aliases = append(aliases, c.Alias)
	}
	return strings.Join(parts, ", "), aliases, nil
}

func buildJoins(joins []domain.ReportJoin) (string, error) {
	sorted := make([]domain.ReportJoin, len(joins))
	copy(sorted, joins)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].JoinOrder < sorted[j].JoinOrder })

	parts := make([]string, 0, len(sorted))
	for _, j := range sorted {
		if err := validIdentifier(j.TableName); err != nil {
			return "", err
		}
		if err := validIdentifier(j.TableAlias); err != nil {
			return "", err
		}
		left, err := qualify(j.LeftAlias, j.LeftColumn)
		if err != nil {
			return "", err
		}
		right, err := qualify(j.RightAlias, j.RightColumn)
		if err != nil {
			return "", err
		}
		joinType := strings.ToUpper(string(j.JoinType))
		switch joinType {
		case "INNER", "LEFT", "RIGHT", "FULL":
		default:
			return "", fmt.Errorf("invalid join type %q", j.JoinType)
		}
		parts = append(parts, fmt.Sprintf("%s JOIN %s AS %s ON %s = %s",
			joinType,
			quoteIdent(j.TableName), quoteIdent(j.TableAlias),
			quoteQualified(left), quoteQualified(right),
		))
	}
	return strings.Join(parts, " "), nil
}

// quoteQualified turns a validated "alias.column" string into
// "alias"."column". Split is safe here because qualify already restricted
// both parts to [A-Za-z_][A-Za-z0-9_]*.
func quoteQualified(aliasDotColumn string) string {
	parts := strings.SplitN(aliasDotColumn, ".", 2)
	return quoteIdent(parts[0]) + "." + quoteIdent(parts[1])
}

func buildWhere(defs []domain.ReportFilter, criteria []domain.FilterCriterion, binder *paramBinder) (string, error) {
	if len(criteria) == 0 {
		return "", nil
	}
	defByField := make(map[string]domain.ReportFilter, len(defs))
	for _, d := range defs {
		defByField[d.FieldName] = d
	}

	parts := make([]string, 0, len(criteria))
	for _, fc := range criteria {
		def, ok := defByField[fc.Field]
		if !ok {
			return "", fmt.Errorf("unknown filter field %q", fc.Field)
		}
		colExpr, err := qualify(def.TableAlias, def.ColumnName)
		if err != nil {
			return "", err
		}
		cond, err := renderCondition(quoteQualified(colExpr), fc.Operator, fc.Value, binder)
		if err != nil {
			return "", fmt.Errorf("field %q: %w", fc.Field, err)
		}
		parts = append(parts, cond)
	}
	return strings.Join(parts, " AND "), nil
}

func buildGroupBy(groups []domain.ReportGroup) (string, error) {
	sorted := make([]domain.ReportGroup, len(groups))
	copy(sorted, groups)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].DisplayOrder < sorted[j].DisplayOrder })

	parts := make([]string, 0, len(sorted))
	for _, g := range sorted {
		q, err := qualify(g.TableAlias, g.ColumnName)
		if err != nil {
			return "", err
		}
		parts = append(parts, quoteQualified(q))
	}
	return strings.Join(parts, ", "), nil
}

func buildOrderBy(defs []domain.ReportSort, criteria []domain.SortCriterion) (string, error) {
	defByField := make(map[string]domain.ReportSort, len(defs))
	for _, d := range defs {
		defByField[d.FieldName] = d
	}

	type entry struct {
		alias, column string
		dir           domain.SortDirection
		priority      int
	}
	var entries []entry

	if len(criteria) > 0 {
		for _, sc := range criteria {
			def, ok := defByField[sc.Field]
			if !ok {
				return "", fmt.Errorf("unknown sort field %q", sc.Field)
			}
			entries = append(entries, entry{def.TableAlias, def.ColumnName, sc.Direction, 0})
		}
	} else {
		sortedDefs := make([]domain.ReportSort, len(defs))
		copy(sortedDefs, defs)
		sort.SliceStable(sortedDefs, func(i, j int) bool { return sortedDefs[i].Priority < sortedDefs[j].Priority })
		for _, d := range sortedDefs {
			entries = append(entries, entry{d.TableAlias, d.ColumnName, d.DefaultDir, d.Priority})
		}
	}

	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		q, err := qualify(e.alias, e.column)
		if err != nil {
			return "", err
		}
		dir := strings.ToUpper(string(e.dir))
		if dir != "ASC" && dir != "DESC" {
			return "", fmt.Errorf("invalid sort direction %q", e.dir)
		}
		parts = append(parts, fmt.Sprintf("%s %s", quoteQualified(q), dir))
	}
	return strings.Join(parts, ", "), nil
}

func buildCountSQL(fromClause, joinClause, whereClause, groupByClause string) string {
	var b strings.Builder
	if groupByClause == "" {
		b.WriteString("SELECT COUNT(*) FROM ")
		b.WriteString(fromClause)
		if joinClause != "" {
			b.WriteString(" ")
			b.WriteString(joinClause)
		}
		if whereClause != "" {
			b.WriteString(" WHERE ")
			b.WriteString(whereClause)
		}
		return b.String()
	}

	b.WriteString("SELECT COUNT(*) FROM (SELECT 1 FROM ")
	b.WriteString(fromClause)
	if joinClause != "" {
		b.WriteString(" ")
		b.WriteString(joinClause)
	}
	if whereClause != "" {
		b.WriteString(" WHERE ")
		b.WriteString(whereClause)
	}
	b.WriteString(" GROUP BY ")
	b.WriteString(groupByClause)
	b.WriteString(") AS sub")
	return b.String()
}
