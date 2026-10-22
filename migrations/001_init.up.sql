CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tasks (
    id          BIGSERIAL PRIMARY KEY,
    owner_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'todo'
        CHECK (status IN ('todo', 'in_progress', 'done')),
    priority    SMALLINT    NOT NULL DEFAULT 2
        CHECK (priority BETWEEN 1 AND 3),
    due_date    TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tasks_owner_status ON tasks (owner_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_owner_priority ON tasks (owner_id, priority DESC, id DESC);
