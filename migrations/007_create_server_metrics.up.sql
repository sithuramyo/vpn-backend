CREATE TABLE server_metrics (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vpn_server_id        UUID NOT NULL REFERENCES vpn_servers (id) ON DELETE CASCADE,
    cpu_usage            DOUBLE PRECISION,
    memory_usage         DOUBLE PRECISION,
    bandwidth_in         BIGINT NOT NULL DEFAULT 0,
    bandwidth_out        BIGINT NOT NULL DEFAULT 0,
    active_connections   INTEGER NOT NULL DEFAULT 0,
    recorded_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_server_metrics_server_id_recorded_at ON server_metrics (vpn_server_id, recorded_at DESC);
