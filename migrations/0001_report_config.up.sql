CREATE TABLE report_templates (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE,
    description   TEXT,
    base_table    TEXT NOT NULL,
    base_alias    TEXT NOT NULL DEFAULT 't0',
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    max_page_size INT NOT NULL DEFAULT 200,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE report_columns (
    id            BIGSERIAL PRIMARY KEY,
    report_id     BIGINT NOT NULL REFERENCES report_templates(id) ON DELETE CASCADE,
    table_alias   TEXT NOT NULL,
    column_name   TEXT NOT NULL,
    alias         TEXT NOT NULL,
    expression    TEXT,
    data_type     TEXT NOT NULL DEFAULT 'string',
    is_visible    BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INT NOT NULL DEFAULT 0,
    UNIQUE (report_id, alias)
);

CREATE TABLE report_joins (
    id            BIGSERIAL PRIMARY KEY,
    report_id     BIGINT NOT NULL REFERENCES report_templates(id) ON DELETE CASCADE,
    join_type     TEXT NOT NULL DEFAULT 'LEFT'
                    CHECK (join_type IN ('INNER','LEFT','RIGHT','FULL')),
    table_name    TEXT NOT NULL,
    table_alias   TEXT NOT NULL,
    left_alias    TEXT NOT NULL,
    left_column   TEXT NOT NULL,
    right_alias   TEXT NOT NULL,
    right_column  TEXT NOT NULL,
    join_order    INT NOT NULL DEFAULT 0
);

CREATE TABLE report_filters (
    id            BIGSERIAL PRIMARY KEY,
    report_id     BIGINT NOT NULL REFERENCES report_templates(id) ON DELETE CASCADE,
    field_name    TEXT NOT NULL,
    table_alias   TEXT NOT NULL,
    column_name   TEXT NOT NULL,
    data_type     TEXT NOT NULL DEFAULT 'string',
    operators     TEXT[] NOT NULL DEFAULT ARRAY['=']::TEXT[],
    required      BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (report_id, field_name)
);

CREATE TABLE report_sorts (
    id            BIGSERIAL PRIMARY KEY,
    report_id     BIGINT NOT NULL REFERENCES report_templates(id) ON DELETE CASCADE,
    field_name    TEXT NOT NULL,
    table_alias   TEXT NOT NULL,
    column_name   TEXT NOT NULL,
    default_dir   TEXT NOT NULL DEFAULT 'asc' CHECK (default_dir IN ('asc','desc')),
    priority      INT NOT NULL DEFAULT 0,
    UNIQUE (report_id, field_name)
);

CREATE TABLE report_groups (
    id            BIGSERIAL PRIMARY KEY,
    report_id     BIGINT NOT NULL REFERENCES report_templates(id) ON DELETE CASCADE,
    table_alias   TEXT NOT NULL,
    column_name   TEXT NOT NULL,
    display_order INT NOT NULL DEFAULT 0
);

CREATE TABLE report_exports (
    id            BIGSERIAL PRIMARY KEY,
    report_id     BIGINT NOT NULL REFERENCES report_templates(id) ON DELETE CASCADE UNIQUE,
    allow_csv     BOOLEAN NOT NULL DEFAULT TRUE,
    allow_excel   BOOLEAN NOT NULL DEFAULT TRUE,
    allow_json    BOOLEAN NOT NULL DEFAULT TRUE,
    max_rows      INT NOT NULL DEFAULT 50000
);

CREATE INDEX idx_report_columns_report_id ON report_columns(report_id);
CREATE INDEX idx_report_joins_report_id ON report_joins(report_id);
CREATE INDEX idx_report_filters_report_id ON report_filters(report_id);
CREATE INDEX idx_report_sorts_report_id ON report_sorts(report_id);
CREATE INDEX idx_report_groups_report_id ON report_groups(report_id);
