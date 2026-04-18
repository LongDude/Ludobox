CREATE TYPE room_status AS ENUM ('open', 'in_game', 'completed');

CREATE OR REPLACE FUNCTION validate_winning_distribution(
    distribution INT[],
    winners_count INT
)
RETURNS BOOLEAN
LANGUAGE SQL
IMMUTABLE
AS $$
    SELECT
        distribution IS NOT NULL
        AND winners_count IS NOT NULL
        AND array_length(distribution, 1) = winners_count
        AND NOT EXISTS (
            SELECT 1
            FROM unnest(distribution) AS item(value)
            WHERE item.value < 0 OR item.value > 100
        )
        AND COALESCE(
            (
                SELECT SUM(item.value)
                FROM unnest(distribution) AS item(value)
            ),
            0
        ) = 100;
$$;

CREATE OR REPLACE FUNCTION prevent_config_mutation_except_archive()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF (to_jsonb(NEW) - 'archived_at') IS DISTINCT FROM (to_jsonb(OLD) - 'archived_at') THEN
        RAISE EXCEPTION 'config rows are append-only; only archived_at may be updated';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TABLE IF NOT EXISTS games
(
    game_id     BIGSERIAL PRIMARY KEY,
    name_game   TEXT NOT NULL,
    archived_at TIMESTAMPTZ NULL,
    CONSTRAINT uq_games_name_game UNIQUE (name_game)
);

CREATE TABLE IF NOT EXISTS game_servers
(
    server_id   BIGSERIAL PRIMARY KEY,
    archived_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS users
(
    user_id  BIGINT PRIMARY KEY,
    nickname TEXT NOT NULL,
    balance  BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0)
);

CREATE TABLE IF NOT EXISTS config
(
    config_id             BIGSERIAL PRIMARY KEY,
    game_id               BIGINT NOT NULL REFERENCES games(game_id),
    capacity              INT NOT NULL CHECK (capacity BETWEEN 2 AND 20),
    registration_price    BIGINT NOT NULL CHECK (registration_price >= 0),
    is_boost              BOOLEAN NOT NULL DEFAULT FALSE,
    boost_price           BIGINT NOT NULL DEFAULT 0 CHECK (boost_price >= 0),
    boost_power           INT NOT NULL DEFAULT 0 CHECK (boost_power BETWEEN 0 AND 100),
    number_winners        INT NOT NULL CHECK (number_winners BETWEEN 1 AND 20),
    winning_distribution  INT[] NOT NULL,
    commission            INT NOT NULL CHECK (commission BETWEEN 0 AND 100),
    time                  INT NOT NULL DEFAULT 60 CHECK (time > 0),
    min_users             INT NOT NULL CHECK (min_users >= 1),
    archived_at           TIMESTAMPTZ NULL,
    CONSTRAINT chk_config_number_winners_capacity CHECK (number_winners <= capacity),
    CONSTRAINT chk_config_min_users_capacity CHECK (min_users <= capacity),
    CONSTRAINT chk_config_boost_when_disabled CHECK (
        is_boost
        OR (boost_price = 0 AND boost_power = 0)
    ),
    CONSTRAINT chk_config_winning_distribution CHECK (
        validate_winning_distribution(winning_distribution, number_winners)
    )
);

CREATE TRIGGER trg_config_append_only
    BEFORE UPDATE ON config
    FOR EACH ROW
    EXECUTE FUNCTION prevent_config_mutation_except_archive();

CREATE TABLE IF NOT EXISTS rooms
(
    room_id      BIGSERIAL PRIMARY KEY,
    config_id    BIGINT NOT NULL REFERENCES config(config_id),
    server_id    BIGINT NOT NULL REFERENCES game_servers(server_id),
    status       room_status NOT NULL DEFAULT 'open',
    archived_at  TIMESTAMPTZ NULL
);

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

CREATE INDEX IF NOT EXISTS idx_rooms_status ON rooms (status);
CREATE INDEX IF NOT EXISTS idx_rooms_config_id ON rooms (config_id);
CREATE INDEX IF NOT EXISTS idx_rooms_server_id ON rooms (server_id);
CREATE INDEX IF NOT EXISTS idx_config_game_id ON config (game_id);
CREATE INDEX IF NOT EXISTS idx_rounds_datetime ON rounds (datetime);
CREATE INDEX IF NOT EXISTS idx_round_participants_room_id ON round_participants (room_id);
CREATE INDEX IF NOT EXISTS idx_round_participants_user_id ON round_participants (user_id);
