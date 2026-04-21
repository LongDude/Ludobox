DROP TRIGGER IF EXISTS trg_notify_user_balance_delete ON users;
DROP TRIGGER IF EXISTS trg_notify_user_balance_update ON users;
DROP TRIGGER IF EXISTS trg_notify_user_balance_insert ON users;

DROP FUNCTION IF EXISTS notify_user_balance_event();
