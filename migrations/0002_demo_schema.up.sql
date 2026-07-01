-- Demo business tables used by the sample "user_directory" report so the
-- service is runnable end-to-end out of the box.

CREATE TABLE departments (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'Active',
    department_id BIGINT REFERENCES departments(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO departments (name) VALUES ('Engineering'), ('Sales'), ('Support');

INSERT INTO users (name, status, department_id, created_at) VALUES
    ('Asha Kapoor', 'Active', 1, '2026-01-15'),
    ('Ben Ortiz', 'Active', 2, '2026-02-20'),
    ('Chen Wei', 'Inactive', 1, '2026-03-05'),
    ('Dana Ivory', 'Active', 3, '2026-04-11'),
    ('Elan Musgrove', 'Active', 2, '2026-05-30');
