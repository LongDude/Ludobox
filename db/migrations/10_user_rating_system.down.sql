DROP INDEX IF EXISTS idx_user_rating_history_rounds_id;
DROP INDEX IF EXISTS idx_user_rating_history_user_created_at;
DROP INDEX IF EXISTS uq_user_rating_history_round_participant_source;

DROP TABLE IF EXISTS user_rating_history;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_rating_check;

ALTER TABLE users
    DROP COLUMN IF EXISTS rating;
