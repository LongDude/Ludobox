package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"user_service/internal/domain"
	"user_service/internal/repository"
)

const gameHistoryResultExpression = `
	CASE
		WHEN rp.winning_money > 0 THEN 'won'
		WHEN rp.exit_room_at IS NOT NULL AND (rd.archived_at IS NULL OR rp.exit_room_at <= rd.archived_at) THEN 'left'
		WHEN rd.status = 'cancelled' THEN 'cancelled'
		WHEN rd.status = 'finished' THEN 'lost'
		ELSE rd.status::TEXT
	END`

func (r *gameHistoryRepository) GetUserGameHistory(ctx context.Context, userID int, params domain.GameHistoryParams) (domain.ListResponse[domain.GameHistoryItem], error) {
	response := domain.ListResponse[domain.GameHistoryItem]{
		Items: make([]domain.GameHistoryItem, 0),
	}
	if err := validateGameHistoryStatus(params.Status); err != nil {
		return response, err
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	whereParts := []string{"rp.user_id = $1"}
	args := []any{userID}

	if params.GameID != nil {
		args = append(args, *params.GameID)
		whereParts = append(whereParts, fmt.Sprintf("g.game_id = $%d", len(args)))
	}
	if params.RoomID != nil {
		args = append(args, *params.RoomID)
		whereParts = append(whereParts, fmt.Sprintf("rd.room_id = $%d", len(args)))
	}
	if params.DateFrom != nil {
		args = append(args, params.DateFrom.UTC())
		whereParts = append(whereParts, fmt.Sprintf("rd.created_at >= $%d", len(args)))
	}
	if params.DateTo != nil {
		args = append(args, params.DateTo.UTC())
		whereParts = append(whereParts, fmt.Sprintf("rd.created_at <= $%d", len(args)))
	}
	if status := strings.ToLower(strings.TrimSpace(params.Status)); status != "" {
		args = append(args, status)
		if status == "finished" {
			whereParts = append(whereParts, fmt.Sprintf("rd.status::TEXT = $%d", len(args)))
		} else {
			whereParts = append(whereParts, fmt.Sprintf("(%s) = $%d", gameHistoryResultExpression, len(args)))
		}
	}

	whereSQL := strings.Join(whereParts, " AND ")
	fromSQL := `
		FROM round_participants rp
		INNER JOIN rounds rd ON rd.rounds_id = rp.rounds_id
		INNER JOIN rooms r ON r.room_id = rd.room_id
		INNER JOIN config c ON c.config_id = r.config_id
		INNER JOIN games g ON g.game_id = c.game_id
		LEFT JOIN (
			SELECT
				round_participants_id,
				COALESCE(SUM(amount) FILTER (WHERE reservation_type = 'entry_fee' AND status <> 'released'), 0) AS entry_fee,
				COALESCE(SUM(amount) FILTER (WHERE reservation_type = 'boost' AND status <> 'released'), 0) AS boost_fee
			FROM user_balance_reservations
			GROUP BY round_participants_id
		) fees ON fees.round_participants_id = rp.round_participants_id
		WHERE ` + whereSQL

	countQuery := `SELECT COUNT(*) ` + fromSQL
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&response.Total); err != nil {
		return response, fmt.Errorf("count user game history: %w", err)
	}

	offset := (page - 1) * pageSize
	listArgs := append(args, pageSize, offset)
	listQuery := fmt.Sprintf(`
		SELECT
			rd.rounds_id,
			rp.round_participants_id,
			rd.room_id,
			g.game_id,
			g.name_game,
			rp.number_in_room,
			rd.status::TEXT,
			%s AS result,
			COALESCE(fees.entry_fee, 0) AS entry_fee,
			COALESCE(fees.boost_fee, 0) AS boost_fee,
			rp.winning_money,
			(rp.winning_money - COALESCE(fees.entry_fee, 0) - COALESCE(fees.boost_fee, 0)) AS net_result,
			rd.created_at,
			rd.archived_at
		%s
		ORDER BY rd.created_at DESC, rp.round_participants_id DESC
		LIMIT $%d OFFSET $%d
	`, gameHistoryResultExpression, fromSQL, len(args)+1, len(args)+2)

	rows, err := r.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return response, fmt.Errorf("list user game history: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanGameHistoryItem(rows)
		if err != nil {
			return response, fmt.Errorf("scan user game history: %w", err)
		}
		response.Items = append(response.Items, item)
	}
	if err := rows.Err(); err != nil {
		return response, fmt.Errorf("iterate user game history: %w", err)
	}

	return response, nil
}

func scanGameHistoryItem(row rowScanner) (domain.GameHistoryItem, error) {
	var item domain.GameHistoryItem
	var archivedAt sql.NullTime

	err := row.Scan(
		&item.RoundID,
		&item.ParticipantID,
		&item.RoomID,
		&item.GameID,
		&item.GameName,
		&item.SeatNumber,
		&item.RoundStatus,
		&item.Result,
		&item.EntryFee,
		&item.BoostFee,
		&item.WinningMoney,
		&item.NetResult,
		&item.JoinedAt,
		&archivedAt,
	)
	if err != nil {
		return item, err
	}

	item.JoinedAt = item.JoinedAt.UTC()
	if archivedAt.Valid {
		finishedAt := archivedAt.Time.UTC()
		item.FinishedAt = &finishedAt
	}

	return item, nil
}

func validateGameHistoryStatus(status string) error {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "won", "lost", "left", "cancelled", "waiting", "active", "finished":
		return nil
	default:
		return fmt.Errorf("%w: unsupported game history status %q", repository.ErrorInvalidListParams, status)
	}
}
