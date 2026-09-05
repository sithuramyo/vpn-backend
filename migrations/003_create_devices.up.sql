CREATE TABLE vpn_devices (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vpn_user_id   UUID NOT NULL REFERENCES vpn_users (id) ON DELETE CASCADE,
    name          VARCHAR(255),
    platform      VARCHAR(20) NOT NULL CHECK (platform IN ('ANDROID', 'IOS', 'WINDOWS', 'MACOS')),
    status        VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED', 'REVOKED')),
    last_seen_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vpn_devices_vpn_user_id ON vpn_devices (vpn_user_id);
CREATE INDEX idx_vpn_devices_status ON vpn_devices (status);
