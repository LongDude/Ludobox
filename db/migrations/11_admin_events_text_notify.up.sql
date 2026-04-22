CREATE OR REPLACE FUNCTION notify_admin_user_service_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    payload_row JSONB;
    payload_id BIGINT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        payload_row := to_jsonb(OLD);
    ELSE
        payload_row := to_jsonb(NEW);
    END IF;

    payload_id := COALESCE((payload_row ->> TG_ARGV[1])::BIGINT, 0);

    PERFORM pg_notify(
        'admin_user_service_events',
        concat_ws(
            '|',
            TG_ARGV[0],
            lower(TG_OP),
            payload_id::TEXT,
            to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
        )
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;

    RETURN NEW;
END;
$$;
