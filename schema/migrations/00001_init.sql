-- +goose Up
CREATE TYPE task_status AS ENUM ('created', 'in_progress', 'completed', 'overdue');

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tasks (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title       text NOT NULL,
    description text NOT NULL DEFAULT '',
    deadline    timestamptz NOT NULL,
    status      task_status NOT NULL DEFAULT 'created',
    creator_id  uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    assignee_id uuid NOT NULL REFERENCES users (id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tasks_assignee_id_idx ON tasks (assignee_id);
CREATE INDEX tasks_creator_id_idx ON tasks (creator_id);
CREATE INDEX tasks_status_idx ON tasks (status);
CREATE INDEX tasks_deadline_idx ON tasks (deadline);

-- +goose Down
DROP TABLE tasks;
DROP TABLE users;
DROP TYPE task_status;
