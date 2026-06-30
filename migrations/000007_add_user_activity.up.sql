ALTER TABLE scams ADD COLUMN IF NOT EXISTS reporter_id UUID REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_scams_reporter_id ON scams(reporter_id);

CREATE TABLE IF NOT EXISTS scam_experiences (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scam_id    UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_hash    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(scam_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_scam_experiences_scam_id ON scam_experiences(scam_id);
CREATE INDEX IF NOT EXISTS idx_scam_experiences_user_id ON scam_experiences(user_id);
