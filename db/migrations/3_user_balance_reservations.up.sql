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
