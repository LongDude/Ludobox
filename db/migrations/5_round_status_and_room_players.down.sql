DROP INDEX IF EXISTS idx_rooms_current_players;
DROP INDEX IF EXISTS idx_rounds_status;

ALTER TABLE rooms
    DROP COLUMN IF EXISTS current_players;

ALTER TABLE rounds
    DROP COLUMN IF EXISTS status;

DROP TYPE IF EXISTS round_status;
