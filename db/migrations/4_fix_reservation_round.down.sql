DROP INDEX IF EXISTS idx_user_balance_reservations_active_expires_at;
DROP INDEX IF EXISTS idx_user_balance_reservations_status_created_at;
DROP INDEX IF EXISTS idx_user_balance_reservations_round_participants_id;

DROP TABLE IF EXISTS user_balance_reservations;

DROP TYPE IF EXISTS reservation_type;
DROP TYPE IF EXISTS reservation_status;

DROP INDEX IF EXISTS idx_round_participants_exit_room_at;
DROP INDEX IF EXISTS idx_round_participants_user_id;
DROP INDEX IF EXISTS idx_round_participants_rounds_id;

DROP TABLE IF EXISTS round_participants;

DROP INDEX IF EXISTS idx_rounds_archived_at;
DROP INDEX IF EXISTS idx_rounds_created_at;
DROP INDEX IF EXISTS idx_rounds_room_id;

DROP TABLE IF EXISTS rounds;

CREATE TABLE IF NOT EXISTS rounds
(
    room_id    BIGINT PRIMARY KEY REFERENCES rooms(room_id) ON DELETE CASCADE,
    datetime   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS round_participants
(
    user_id        BIGINT NOT NULL REFERENCES users(user_id),
    room_id        BIGINT NOT NULL REFERENCES rounds(room_id) ON DELETE CASCADE,
    winning_money  BIGINT NOT NULL DEFAULT 0 CHECK (winning_money >= 0),
    PRIMARY KEY (user_id, room_id)
);

CREATE INDEX IF NOT EXISTS idx_rounds_datetime ON rounds (datetime);
CREATE INDEX IF NOT EXISTS idx_round_participants_room_id ON round_participants (room_id);
CREATE INDEX IF NOT EXISTS idx_round_participants_user_id ON round_participants (user_id);

CREATE TYPE reservation_status AS ENUM ('active', 'committed', 'released', 'expired');

CREATE TYPE reservation_type AS ENUM ('entry_fee', 'boost');

CREATE OR REPLACE FUNCTION set_user_balance_reservations_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TABLE IF NOT EXISTS user_balance_reservations
(
    reservation_id   BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL REFERENCES users(user_id),
    room_id          BIGINT NOT NULL REFERENCES rooms(room_id),
    reservation_type reservation_type NOT NULL,
    amount           BIGINT NOT NULL CHECK (amount > 0),
    status           reservation_status NOT NULL DEFAULT 'active',
    idempotency_key  TEXT NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    committed_at     TIMESTAMPTZ NULL,
    released_at      TIMESTAMPTZ NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at      TIMESTAMPTZ NULL,
    CONSTRAINT uq_user_balance_reservations_idempotency_key UNIQUE (idempotency_key),
    CONSTRAINT chk_user_balance_reservations_resolution_timestamps CHECK (
        (status = 'active' AND committed_at IS NULL AND released_at IS NULL)
        OR (status = 'committed' AND committed_at IS NOT NULL AND released_at IS NULL)
        OR (status IN ('released', 'expired') AND committed_at IS NULL AND released_at IS NOT NULL)
    ),
    CONSTRAINT chk_user_balance_reservations_expiration_order CHECK (
        expires_at >= created_at
    )
);

CREATE TRIGGER trg_user_balance_reservations_updated_at
    BEFORE UPDATE ON user_balance_reservations
    FOR EACH ROW
    EXECUTE FUNCTION set_user_balance_reservations_updated_at();

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_balance_reservations_active_per_type
    ON user_balance_reservations (user_id, room_id, reservation_type)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_user_balance_reservations_user_status
    ON user_balance_reservations (user_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_balance_reservations_room_status
    ON user_balance_reservations (room_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_balance_reservations_active_expires_at
    ON user_balance_reservations (expires_at)
    WHERE status = 'active';
