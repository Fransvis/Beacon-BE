CREATE TABLE IF NOT EXISTS scammer_names (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scam_id    UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scammer_names_scam_id ON scammer_names(scam_id);
CREATE INDEX IF NOT EXISTS idx_scammer_names_name ON scammer_names(name);

ALTER TABLE scams DROP COLUMN IF EXISTS scammer_names;
