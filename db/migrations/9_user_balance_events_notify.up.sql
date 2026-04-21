CREATE OR REPLACE FUNCTION notify_user_balance_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM pg_notify(
            'user_balance_events',
            json_build_object(
                'type', 'user_balance_changed',
                'action', 'delete',
                'user_id', OLD.user_id,
                'balance', 0,
                'timestamp', to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
            )::TEXT
        );

        RETURN OLD;
    END IF;

    PERFORM pg_notify(
        'user_balance_events',
        json_build_object(
            'type', 'user_balance_changed',
            'action', lower(TG_OP),
            'user_id', NEW.user_id,
            'balance', NEW.balance,
            'timestamp', to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
        )::TEXT
    );

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_notify_user_balance_insert ON users;
CREATE TRIGGER trg_notify_user_balance_insert
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION notify_user_balance_event();

DROP TRIGGER IF EXISTS trg_notify_user_balance_update ON users;
CREATE TRIGGER trg_notify_user_balance_update
    AFTER UPDATE OF balance ON users
    FOR EACH ROW
    WHEN (OLD.balance IS DISTINCT FROM NEW.balance)
    EXECUTE FUNCTION notify_user_balance_event();

DROP TRIGGER IF EXISTS trg_notify_user_balance_delete ON users;
CREATE TRIGGER trg_notify_user_balance_delete
    AFTER DELETE ON users
    FOR EACH ROW
    EXECUTE FUNCTION notify_user_balance_event();
