DROP INDEX IF EXISTS idx_game_servers_status_last_heartbeat;
DROP INDEX IF EXISTS uq_game_servers_instance_key;

ALTER TABLE IF EXISTS game_servers
    DROP CONSTRAINT IF EXISTS chk_game_servers_status;

ALTER TABLE IF EXISTS game_servers
    DROP COLUMN IF EXISTS last_heartbeat_at;

ALTER TABLE IF EXISTS game_servers
    DROP COLUMN IF EXISTS started_at;

ALTER TABLE IF EXISTS game_servers
    DROP COLUMN IF EXISTS status;

ALTER TABLE IF EXISTS game_servers
    DROP COLUMN IF EXISTS redis_host;

ALTER TABLE IF EXISTS game_servers
    DROP COLUMN IF EXISTS instance_key;
