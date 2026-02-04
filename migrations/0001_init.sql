CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS posts (
  id UUID PRIMARY KEY,
  title TEXT NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  content_md TEXT NOT NULL DEFAULT '',
  tags TEXT[] NOT NULL DEFAULT '{}',
  author_name TEXT NOT NULL,
  author_nickname TEXT NOT NULL,
  status TEXT NOT NULL,
  scheduled_for TIMESTAMPTZ,
  published_at TIMESTAMPTZ,
  model TEXT NOT NULL DEFAULT '',
  prompt_version TEXT NOT NULL DEFAULT 'v1',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_posts_status_scheduled ON posts (status, scheduled_for);
CREATE INDEX IF NOT EXISTS idx_posts_published_at ON posts (published_at DESC);
