CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL               PRIMARY KEY,
    username        TEXT        NOT NULL    UNIQUE,
    email           TEXT        NOT NULL    UNIQUE,
    password_hash   TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL    DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounts (
    id              BIGSERIAL                 PRIMARY KEY,
    user_id         BIGINT           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance         NUMERIC(14, 2)   NOT NULL DEFAULT 0 CHECK (balance >= 0),
    currency        TEXT             NOT NULL DEFAULT 'RUB',
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);