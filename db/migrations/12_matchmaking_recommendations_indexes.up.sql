CREATE INDEX IF NOT EXISTS idx_round_participants_active_rounds_id
    ON round_participants (rounds_id)
    WHERE exit_room_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_round_participants_active_user_rounds_id
    ON round_participants (user_id, rounds_id)
    WHERE exit_room_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_rooms_open_server_config
    ON rooms (server_id, config_id, room_id)
    WHERE archived_at IS NULL AND status = 'open';

CREATE INDEX IF NOT EXISTS idx_game_servers_up_heartbeat
    ON game_servers (last_heartbeat_at DESC, server_id)
    WHERE status = 'up' AND archived_at IS NULL;
