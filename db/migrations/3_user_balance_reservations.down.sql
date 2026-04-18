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
