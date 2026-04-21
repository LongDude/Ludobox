DROP TRIGGER IF EXISTS trg_admin_notify_game_servers ON game_servers;
DROP TRIGGER IF EXISTS trg_admin_notify_rooms ON rooms;
DROP TRIGGER IF EXISTS trg_admin_notify_config ON config;
DROP TRIGGER IF EXISTS trg_admin_notify_games ON games;

DROP FUNCTION IF EXISTS notify_admin_user_service_event();
