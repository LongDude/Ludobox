DROP INDEX IF EXISTS idx_round_participants_user_id;
DROP INDEX IF EXISTS idx_round_participants_room_id;
DROP INDEX IF EXISTS idx_rounds_datetime;
DROP INDEX IF EXISTS idx_config_game_id;
DROP INDEX IF EXISTS idx_rooms_server_id;
DROP INDEX IF EXISTS idx_rooms_config_id;
DROP INDEX IF EXISTS idx_rooms_status;

DROP TABLE IF EXISTS round_participants;
DROP TABLE IF EXISTS rounds;
DROP TABLE IF EXISTS rooms;

DO $$
BEGIN
    IF to_regclass('config') IS NOT NULL THEN
        EXECUTE 'DROP TRIGGER IF EXISTS trg_config_append_only ON config';
    END IF;
END;
$$;

DROP TABLE IF EXISTS config;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS game_servers;
DROP TABLE IF EXISTS games;

DROP FUNCTION IF EXISTS prevent_config_mutation_except_archive();
DROP FUNCTION IF EXISTS validate_winning_distribution(INT[], INT);

DROP TYPE IF EXISTS room_status;
