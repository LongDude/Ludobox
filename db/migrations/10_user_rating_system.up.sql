ALTER TABLE users
    ADD COLUMN IF NOT EXISTS rating BIGINT NOT NULL DEFAULT 0 CHECK (rating >= 0);

CREATE TABLE IF NOT EXISTS user_rating_history
(
    user_rating_history_id BIGSERIAL PRIMARY KEY,
    user_id                BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    round_participants_id  BIGINT NULL REFERENCES round_participants(round_participants_id) ON DELETE SET NULL,
    rounds_id              BIGINT NULL REFERENCES rounds(rounds_id) ON DELETE SET NULL,
    room_id                BIGINT NULL REFERENCES rooms(room_id) ON DELETE SET NULL,
    game_id                BIGINT NULL REFERENCES games(game_id) ON DELETE SET NULL,
    source                 TEXT NOT NULL,
    delta                  BIGINT NOT NULL CHECK (delta >= 0),
    rating_after           BIGINT NOT NULL CHECK (rating_after >= 0),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_rating_history_round_participant_source
    ON user_rating_history (round_participants_id, source)
    WHERE round_participants_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_rating_history_user_created_at
    ON user_rating_history (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_rating_history_rounds_id
    ON user_rating_history (rounds_id);
