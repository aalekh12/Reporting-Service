-- Sample report config equivalent to the assignment's example SQL:
--   SELECT u.id, u.name, d.name department
--   FROM users u LEFT JOIN departments d ON u.department_id = d.id
--   WHERE u.status = $1 AND u.created_at BETWEEN $2 AND $3
--   ORDER BY u.created_at DESC LIMIT 50 OFFSET 0;

WITH template AS (
    INSERT INTO report_templates (name, description, base_table, base_alias, enabled, max_page_size)
    VALUES ('user_directory', 'Users with their department', 'users', 'u', TRUE, 200)
    RETURNING id
)
INSERT INTO report_columns (report_id, table_alias, column_name, alias, data_type, is_visible, display_order)
SELECT id, 'u', 'id', 'id', 'int', TRUE, 1 FROM template
UNION ALL
SELECT id, 'u', 'name', 'name', 'string', TRUE, 2 FROM template
UNION ALL
SELECT id, 'd', 'name', 'department', 'string', TRUE, 3 FROM template
UNION ALL
SELECT id, 'u', 'status', 'status', 'string', TRUE, 4 FROM template
UNION ALL
SELECT id, 'u', 'created_at', 'created_at', 'datetime', TRUE, 5 FROM template;

INSERT INTO report_joins (report_id, join_type, table_name, table_alias, left_alias, left_column, right_alias, right_column, join_order)
SELECT id, 'LEFT', 'departments', 'd', 'u', 'department_id', 'd', 'id', 1
FROM report_templates WHERE name = 'user_directory';

INSERT INTO report_filters (report_id, field_name, table_alias, column_name, data_type, operators, required)
SELECT id, 'status', 'u', 'status', 'string', ARRAY['=','in']::TEXT[], FALSE
FROM report_templates WHERE name = 'user_directory'
UNION ALL
SELECT id, 'created_at', 'u', 'created_at', 'datetime', ARRAY['between','>=','<=']::TEXT[], FALSE
FROM report_templates WHERE name = 'user_directory'
UNION ALL
SELECT id, 'department', 'd', 'name', 'string', ARRAY['=','contains']::TEXT[], FALSE
FROM report_templates WHERE name = 'user_directory';

INSERT INTO report_sorts (report_id, field_name, table_alias, column_name, default_dir, priority)
SELECT id, 'created_at', 'u', 'created_at', 'desc', 1
FROM report_templates WHERE name = 'user_directory'
UNION ALL
SELECT id, 'name', 'u', 'name', 'asc', 2
FROM report_templates WHERE name = 'user_directory';

INSERT INTO report_exports (report_id, allow_csv, allow_excel, allow_json, max_rows)
SELECT id, TRUE, TRUE, TRUE, 50000
FROM report_templates WHERE name = 'user_directory';
