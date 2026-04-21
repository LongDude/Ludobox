DROP INDEX IF EXISTS idx_room_events_rounds_id_created_at;
DROP INDEX IF EXISTS idx_room_events_room_id_created_at;
DROP TABLE IF EXISTS room_events;

ALTER TABLE config
    DROP COLUMN IF EXISTS next_round_delay,
    DROP COLUMN IF EXISTS round_time;
