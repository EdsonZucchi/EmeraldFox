CREATE TABLE IF NOT EXISTS users (
    id         UUID         PRIMARY KEY,
    name       VARCHAR(120) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT users_email_key UNIQUE (email)
);

CREATE INDEX IF NOT EXISTS users_created_at_idx ON users (created_at DESC);
