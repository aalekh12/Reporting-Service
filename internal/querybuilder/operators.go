package querybuilder

import (
	"fmt"

	"reporting-service/internal/domain"
)

// paramBinder accumulates query parameters and hands out "$N" placeholders,
// which is the only mechanism this package uses to get values into SQL.
type paramBinder struct {
	args []any
}

func (p *paramBinder) bind(v any) string {
	p.args = append(p.args, v)
	return fmt.Sprintf("$%d", len(p.args))
}

// knownOperators is the fixed, hardcoded set of operators the engine can
// render. A ReportFilter.Operators list is itself validated against this
// set (see Validator), so a request can only ever select one of these —
// never arbitrary SQL.
var knownOperators = map[domain.Operator]bool{
	domain.OpEqual:       true,
	domain.OpNotEqual:    true,
	domain.OpGreaterThan: true,
	domain.OpGreaterEq:   true,
	domain.OpLessThan:    true,
	domain.OpLessEq:      true,
	domain.OpLike:        true,
	domain.OpContains:    true,
	domain.OpIn:          true,
	domain.OpBetween:     true,
	domain.OpIsNull:      true,
	domain.OpIsNotNull:   true,
}

func isKnownOperator(op domain.Operator) bool {
	return knownOperators[op]
}

// renderCondition renders one WHERE fragment for colExpr <op> value,
// binding value(s) as parameters via binder.
func renderCondition(colExpr string, op domain.Operator, value any, binder *paramBinder) (string, error) {
	switch op {
	case domain.OpEqual:
		return fmt.Sprintf("%s = %s", colExpr, binder.bind(value)), nil
	case domain.OpNotEqual:
		return fmt.Sprintf("%s != %s", colExpr, binder.bind(value)), nil
	case domain.OpGreaterThan:
		return fmt.Sprintf("%s > %s", colExpr, binder.bind(value)), nil
	case domain.OpGreaterEq:
		return fmt.Sprintf("%s >= %s", colExpr, binder.bind(value)), nil
	case domain.OpLessThan:
		return fmt.Sprintf("%s < %s", colExpr, binder.bind(value)), nil
	case domain.OpLessEq:
		return fmt.Sprintf("%s <= %s", colExpr, binder.bind(value)), nil
	case domain.OpLike:
		return fmt.Sprintf("%s LIKE %s", colExpr, binder.bind(value)), nil
	case domain.OpContains:
		s, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("contains requires a string value")
		}
		return fmt.Sprintf("%s ILIKE %s", colExpr, binder.bind("%"+s+"%")), nil
	case domain.OpIn:
		values, err := toSlice(value)
		if err != nil {
			return "", err
		}
		if len(values) == 0 {
			return "", fmt.Errorf("in requires at least one value")
		}
		placeholders := make([]string, len(values))
		for i, v := range values {
			placeholders[i] = binder.bind(v)
		}
		return fmt.Sprintf("%s IN (%s)", colExpr, joinComma(placeholders)), nil
	case domain.OpBetween:
		values, err := toSlice(value)
		if err != nil {
			return "", err
		}
		if len(values) != 2 {
			return "", fmt.Errorf("between requires exactly two values")
		}
		return fmt.Sprintf("%s BETWEEN %s AND %s", colExpr, binder.bind(values[0]), binder.bind(values[1])), nil
	case domain.OpIsNull:
		return fmt.Sprintf("%s IS NULL", colExpr), nil
	case domain.OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", colExpr), nil
	default:
		return "", fmt.Errorf("unsupported operator %q", op)
	}
}

func toSlice(value any) ([]any, error) {
	switch v := value.(type) {
	case []any:
		return v, nil
	case []string:
		out := make([]any, len(v))
		for i, s := range v {
			out[i] = s
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected a list value, got %T", value)
	}
}

func joinComma(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
