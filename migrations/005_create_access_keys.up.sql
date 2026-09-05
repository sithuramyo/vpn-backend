CREATE TABLE access_keys (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vpn_user_id            UUID NOT NULL REFERENCES vpn_users (id) ON DELETE CASCADE,
    vpn_server_id          UUID NOT NULL REFERENCES vpn_servers (id) ON DELETE RESTRICT,
    name                   VARCHAR(255),
    secret_reference       VARCHAR(255) NOT NULL,
    encrypted_secret       TEXT NOT NULL,
    port                   INTEGER NOT NULL,
    cipher                 VARCHAR(64),
    protocol               VARCHAR(20) NOT NULL DEFAULT 'SHADOWSOCKS',
    tcp_enabled            BOOLEAN NOT NULL DEFAULT true,
    udp_enabled            BOOLEAN NOT NULL DEFAULT true,
    websocket_enabled      BOOLEAN NOT NULL DEFAULT true,
    websocket_path         VARCHAR(255),
    websocket_udp_path     VARCHAR(255),
    expires_at             TIMESTAMPTZ,
    traffic_limit_bytes    BIGINT NOT NULL DEFAULT 0,
    traffic_used_bytes     BIGINT NOT NULL DEFAULT 0,
    status                 VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE', 'REVOKED', 'EXPIRED')),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_access_keys_vpn_user_id ON access_keys (vpn_user_id);
CREATE INDEX idx_access_keys_vpn_server_id ON access_keys (vpn_server_id);
CREATE INDEX idx_access_keys_status ON access_keys (status);
