ALTER TABLE config
    ADD COLUMN IF NOT EXISTS round_time INT NOT NULL DEFAULT 60 CHECK (round_time > 0),
    ADD COLUMN IF NOT EXISTS next_round_delay INT NOT NULL DEFAULT 0 CHECK (next_round_delay >= 0);

UPDATE config
SET round_time = time
WHERE round_time = 60;

CREATE TABLE IF NOT EXISTS room_events
(
    room_event_id BIGSERIAL PRIMARY KEY,
    room_id       BIGINT NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    rounds_id     BIGINT NULL REFERENCES rounds(rounds_id) ON DELETE SET NULL,
    event_type    TEXT NOT NULL,
    event_data    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_room_events_room_id_created_at
    ON room_events (room_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_room_events_rounds_id_created_at
    ON room_events (rounds_id, created_at DESC);
