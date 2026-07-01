package domain

// ReportTemplate is the root configuration for a single report.
type ReportTemplate struct {
	ID          int64
	Name        string
	Description string
	BaseTable   string
	BaseAlias   string
	Enabled     bool
	MaxPageSize int

	Columns []ReportColumn
	Joins   []ReportJoin
	Filters []ReportFilter
	Sorts   []ReportSort
	Groups  []ReportGroup
	Export  ReportExport
}

// ReportColumn describes one selected output column.
type ReportColumn struct {
	ID           int64
	ReportID     int64
	TableAlias   string
	ColumnName   string
	Alias        string
	Expression   string // optional raw SQL expression, e.g. "SUM(t1.amount)"
	DataType     DataType
	IsVisible    bool
	DisplayOrder int
}

// ReportJoin describes one join clause against the base table (or another
// already-joined alias).
type ReportJoin struct {
	ID          int64
	ReportID    int64
	JoinType    JoinType
	TableName   string
	TableAlias  string
	LeftAlias   string
	LeftColumn  string
	RightAlias  string
	RightColumn string
	JoinOrder   int
}

// ReportFilter whitelists a field that callers may filter on at runtime.
type ReportFilter struct {
	ID         int64
	ReportID   int64
	FieldName  string
	TableAlias string
	ColumnName string
	DataType   DataType
	Operators  []Operator
	Required   bool
}

// ReportSort whitelists a field that callers may sort on at runtime.
type ReportSort struct {
	ID         int64
	ReportID   int64
	FieldName  string
	TableAlias string
	ColumnName string
	DefaultDir SortDirection
	Priority   int
}

// ReportGroup describes one GROUP BY clause entry.
type ReportGroup struct {
	ID           int64
	ReportID     int64
	TableAlias   string
	ColumnName   string
	DisplayOrder int
}

// ReportExport controls which export formats/limits apply to a report.
type ReportExport struct {
	ID         int64
	ReportID   int64
	AllowCSV   bool
	AllowExcel bool
	AllowJSON  bool
	MaxRows    int
}

type DataType string

const (
	DataTypeString   DataType = "string"
	DataTypeInt      DataType = "int"
	DataTypeFloat    DataType = "float"
	DataTypeBool     DataType = "bool"
	DataTypeDate     DataType = "date"
	DataTypeDateTime DataType = "datetime"
)

type JoinType string

const (
	JoinInner JoinType = "INNER"
	JoinLeft  JoinType = "LEFT"
	JoinRight JoinType = "RIGHT"
	JoinFull  JoinType = "FULL"
)

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type Operator string

const (
	OpEqual       Operator = "="
	OpNotEqual    Operator = "!="
	OpGreaterThan Operator = ">"
	OpGreaterEq   Operator = ">="
	OpLessThan    Operator = "<"
	OpLessEq      Operator = "<="
	OpLike        Operator = "like"
	OpContains    Operator = "contains"
	OpIn          Operator = "in"
	OpBetween     Operator = "between"
	OpIsNull      Operator = "is_null"
	OpIsNotNull   Operator = "is_not_null"
)

// ExportFormat is a requested output format for /export.
type ExportFormat string

const (
	FormatJSON  ExportFormat = "json"
	FormatCSV   ExportFormat = "csv"
	FormatExcel ExportFormat = "xlsx"
)
