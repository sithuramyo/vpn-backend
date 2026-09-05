CREATE TABLE vpn_servers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    hostname    VARCHAR(255) NOT NULL,
    public_ip   INET,
    country     VARCHAR(100),
    city        VARCHAR(100),
    status      VARCHAR(20) NOT NULL CHECK (status IN ('ONLINE', 'OFFLINE', 'DEGRADED', 'MAINTENANCE')),
    vpn_port    INTEGER,
    tls_port    INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
