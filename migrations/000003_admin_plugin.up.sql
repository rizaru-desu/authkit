-- Better Auth admin plugin: ban fields on users, impersonation on sessions.
ALTER TABLE users ADD COLUMN IF NOT EXISTS banned      BOOLEAN     NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS ban_reason  TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS ban_expires TIMESTAMPTZ;

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS impersonated_by TEXT;
