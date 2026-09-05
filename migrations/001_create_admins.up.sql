CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE admins (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub     VARCHAR(255) UNIQUE NOT NULL,
    email          VARCHAR(255) NOT NULL,
    name           VARCHAR(255),
    picture_url    TEXT,
    role           VARCHAR(20) NOT NULL CHECK (role IN ('ADMIN', 'OPERATOR', 'VIEWER')),
    status         VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at  TIMESTAMPTZ
);

CREATE INDEX idx_admins_email ON admins (email);
