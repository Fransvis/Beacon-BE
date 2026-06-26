CREATE TABLE IF NOT EXISTS comments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scam_id       UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    author_name   TEXT,
    content       TEXT NOT NULL,
    is_anonymous  BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_comments_scam_id ON comments(scam_id);
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC);
