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
        json_build_object(
            'type', 'admin_resource_changed',
            'resource', TG_ARGV[0],
            'action', lower(TG_OP),
            'id', payload_id,
            'timestamp', to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
        )::TEXT
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_admin_notify_games ON games;
CREATE TRIGGER trg_admin_notify_games
    AFTER INSERT OR UPDATE OR DELETE ON games
    FOR EACH ROW
    EXECUTE FUNCTION notify_admin_user_service_event('games', 'game_id');

DROP TRIGGER IF EXISTS trg_admin_notify_config ON config;
CREATE TRIGGER trg_admin_notify_config
    AFTER INSERT OR UPDATE OR DELETE ON config
    FOR EACH ROW
    EXECUTE FUNCTION notify_admin_user_service_event('configs', 'config_id');

DROP TRIGGER IF EXISTS trg_admin_notify_rooms ON rooms;
CREATE TRIGGER trg_admin_notify_rooms
    AFTER INSERT OR UPDATE OR DELETE ON rooms
    FOR EACH ROW
    EXECUTE FUNCTION notify_admin_user_service_event('rooms', 'room_id');

DROP TRIGGER IF EXISTS trg_admin_notify_game_servers ON game_servers;
CREATE TRIGGER trg_admin_notify_game_servers
    AFTER INSERT OR UPDATE OR DELETE ON game_servers
    FOR EACH ROW
    EXECUTE FUNCTION notify_admin_user_service_event('servers', 'server_id');
