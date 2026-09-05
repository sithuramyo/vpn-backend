CREATE TABLE audit_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id       UUID REFERENCES admins (id) ON DELETE SET NULL,
    action         VARCHAR(64) NOT NULL,
    resource_type  VARCHAR(64),
    resource_id    UUID,
    metadata       JSONB,
    ip_address     INET,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_admin_id ON audit_logs (admin_id);
CREATE INDEX idx_audit_logs_action ON audit_logs (action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
