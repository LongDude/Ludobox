CREATE TYPE round_status AS ENUM ('waiting', 'active', 'finished', 'cancelled');

ALTER TABLE rounds
    ADD COLUMN IF NOT EXISTS status round_status NOT NULL DEFAULT 'waiting';

ALTER TABLE rooms
    ADD COLUMN IF NOT EXISTS current_players INT NOT NULL DEFAULT 0 CHECK (current_players >= 0);

CREATE INDEX IF NOT EXISTS idx_rounds_status ON rounds (status);
CREATE INDEX IF NOT EXISTS idx_rooms_current_players ON rooms (current_players);
