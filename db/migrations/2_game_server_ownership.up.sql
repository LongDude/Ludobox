ALTER TABLE game_servers
    ADD COLUMN IF NOT EXISTS instance_key TEXT,
    ADD COLUMN IF NOT EXISTS redis_host TEXT,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'down',
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS last_heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE game_servers
SET instance_key = 'legacy-' || server_id
WHERE instance_key IS NULL;

UPDATE game_servers
SET redis_host = 'unknown'
WHERE redis_host IS NULL;

ALTER TABLE game_servers
    ALTER COLUMN instance_key SET NOT NULL;

ALTER TABLE game_servers
    ALTER COLUMN redis_host SET NOT NULL;

ALTER TABLE game_servers
    DROP CONSTRAINT IF EXISTS chk_game_servers_status;

ALTER TABLE game_servers
    ADD CONSTRAINT chk_game_servers_status CHECK (status IN ('up', 'down'));

CREATE UNIQUE INDEX IF NOT EXISTS uq_game_servers_instance_key
    ON game_servers (instance_key);

CREATE INDEX IF NOT EXISTS idx_game_servers_status_last_heartbeat
    ON game_servers (status, last_heartbeat_at DESC);
