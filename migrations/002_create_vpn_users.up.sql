CREATE TABLE vpn_users (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                 VARCHAR(255),
    name                  VARCHAR(255),
    status                VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED', 'EXPIRED')),
    expires_at            TIMESTAMPTZ,
    traffic_limit_bytes   BIGINT NOT NULL DEFAULT 0,
    traffic_used_bytes    BIGINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at            TIMESTAMPTZ
);

CREATE INDEX idx_vpn_users_email ON vpn_users (email);
CREATE INDEX idx_vpn_users_status ON vpn_users (status);
CREATE INDEX idx_vpn_users_deleted_at ON vpn_users (deleted_at);
