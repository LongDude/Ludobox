DROP INDEX IF EXISTS idx_user_balance_reservations_active_expires_at;
DROP INDEX IF EXISTS idx_user_balance_reservations_room_status;
DROP INDEX IF EXISTS idx_user_balance_reservations_user_status;
DROP INDEX IF EXISTS uq_user_balance_reservations_active_per_type;

DO $$
BEGIN
    IF to_regclass('user_balance_reservations') IS NOT NULL THEN
        EXECUTE 'DROP TRIGGER IF EXISTS trg_user_balance_reservations_updated_at ON user_balance_reservations';
    END IF;
END;
$$;

DROP TABLE IF EXISTS user_balance_reservations;
DROP FUNCTION IF EXISTS set_user_balance_reservations_updated_at();
DROP TYPE IF EXISTS reservation_type;
DROP TYPE IF EXISTS reservation_status;

DROP INDEX IF EXISTS idx_round_participants_room_id;
DROP INDEX IF EXISTS idx_round_participants_user_id;
DROP INDEX IF EXISTS idx_rounds_datetime;

DROP TABLE IF EXISTS round_participants;
DROP TABLE IF EXISTS rounds;

CREATE TABLE IF NOT EXISTS rounds
(
    rounds_id    BIGSERIAL PRIMARY KEY,
    room_id      BIGINT NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at  TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_rounds_room_id ON rounds (room_id);
CREATE INDEX IF NOT EXISTS idx_rounds_created_at ON rounds (created_at);
CREATE INDEX IF NOT EXISTS idx_rounds_archived_at ON rounds (archived_at);

CREATE TABLE IF NOT EXISTS round_participants
(
    round_participants_id BIGSERIAL PRIMARY KEY,
    user_id               BIGINT NOT NULL REFERENCES users(user_id),
    rounds_id             BIGINT NOT NULL REFERENCES rounds(rounds_id) ON DELETE CASCADE,
    boost                 INT NOT NULL DEFAULT 0 CHECK (boost >= 0),
    winning_money         BIGINT NOT NULL DEFAULT 0 CHECK (winning_money >= 0),
    number_in_room        INT NOT NULL CHECK (number_in_room > 0),
    exit_room_at          TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_round_participants_rounds_id ON round_participants (rounds_id);
CREATE INDEX IF NOT EXISTS idx_round_participants_user_id ON round_participants (user_id);
CREATE INDEX IF NOT EXISTS idx_round_participants_exit_room_at ON round_participants (exit_room_at);

CREATE TYPE reservation_status AS ENUM ('active', 'released', 'committed');

CREATE TYPE reservation_type AS ENUM ('entry_fee', 'boost');

CREATE TABLE IF NOT EXISTS user_balance_reservations
(
    reservation_id         BIGSERIAL PRIMARY KEY,
    round_participants_id  BIGINT NOT NULL REFERENCES round_participants(round_participants_id) ON DELETE CASCADE,
    reservation_type       reservation_type NOT NULL,
    amount                 BIGINT NOT NULL CHECK (amount > 0),
    status                 reservation_status NOT NULL DEFAULT 'active',
    expires_at             TIMESTAMPTZ NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at            TIMESTAMPTZ NULL,
    CONSTRAINT chk_user_balance_reservations_expiration_order CHECK (expires_at >= created_at)
);

CREATE INDEX IF NOT EXISTS idx_user_balance_reservations_round_participants_id
    ON user_balance_reservations (round_participants_id);

CREATE INDEX IF NOT EXISTS idx_user_balance_reservations_status_created_at
    ON user_balance_reservations (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_balance_reservations_active_expires_at
    ON user_balance_reservations (expires_at)
    WHERE status = 'active';
