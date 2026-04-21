CREATE UNIQUE INDEX IF NOT EXISTS uq_rounds_active_room
    ON rounds (room_id)
    WHERE archived_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_round_participants_active_seat
    ON round_participants (rounds_id, number_in_room)
    WHERE exit_room_at IS NULL;
