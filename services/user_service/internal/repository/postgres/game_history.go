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
		WHEN item.winning_money > 0 THEN 'won'
		WHEN item.has_left AND item.reserved_seats_count = 0 THEN 'left'
		WHEN item.round_status = 'cancelled' THEN 'cancelled'
		WHEN item.round_status = 'finished' THEN 'lost'
		ELSE item.round_status
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

	whereSQL := strings.Join(whereParts, " AND ")
	baseCTE := `
		WITH participant_fees AS (
			SELECT
				round_participants_id,
				COALESCE(SUM(amount) FILTER (WHERE reservation_type = 'entry_fee' AND status IN ('active', 'committed')), 0) AS entry_fee,
				COALESCE(SUM(amount) FILTER (WHERE reservation_type = 'boost' AND status IN ('active', 'committed')), 0) AS boost_fee
			FROM user_balance_reservations
			GROUP BY round_participants_id
		),
		user_round_history AS (
			SELECT
				rd.rounds_id,
				rd.room_id,
				g.game_id,
				g.name_game,
				rd.status::TEXT AS round_status,
				COALESCE(
					array_agg(rp.number_in_room ORDER BY rp.number_in_room)
						FILTER (WHERE rp.exit_room_at IS NULL),
					'{}'::INT[]
				) AS reserved_seats,
				COALESCE(
					array_agg(rp.number_in_room ORDER BY rp.number_in_room)
						FILTER (WHERE rp.exit_room_at IS NULL AND rp.winning_money > 0),
					'{}'::INT[]
				) AS winning_seats,
				COUNT(*) FILTER (WHERE rp.exit_room_at IS NULL)::INT AS reserved_seats_count,
				COUNT(*) FILTER (WHERE rp.exit_room_at IS NULL AND rp.winning_money > 0)::INT AS winning_seats_count,
				COALESCE(SUM(fees.entry_fee), 0) AS entry_fee,
				COALESCE(SUM(fees.boost_fee), 0) AS boost_fee,
				COALESCE(SUM(rp.winning_money) FILTER (WHERE rp.exit_room_at IS NULL), 0) AS winning_money,
				MIN(rd.created_at) AS joined_at,
				MAX(rd.archived_at) AS finished_at,
				BOOL_OR(rp.exit_room_at IS NOT NULL AND (rd.archived_at IS NULL OR rp.exit_room_at <= rd.archived_at)) AS has_left
			FROM round_participants rp
			INNER JOIN rounds rd ON rd.rounds_id = rp.rounds_id
			INNER JOIN rooms r ON r.room_id = rd.room_id
			INNER JOIN config c ON c.config_id = r.config_id
			INNER JOIN games g ON g.game_id = c.game_id
			LEFT JOIN participant_fees fees ON fees.round_participants_id = rp.round_participants_id
			WHERE ` + whereSQL + `
			GROUP BY rd.rounds_id, rd.room_id, g.game_id, g.name_game, rd.status
		)
	`
	filteredWhereParts := make([]string, 0, 1)
	filteredArgs := append([]any(nil), args...)
	if status := strings.ToLower(strings.TrimSpace(params.Status)); status != "" {
		filteredArgs = append(filteredArgs, status)
		if status == "finished" {
			filteredWhereParts = append(filteredWhereParts, fmt.Sprintf("item.round_status = $%d", len(filteredArgs)))
		} else {
			filteredWhereParts = append(filteredWhereParts, fmt.Sprintf("(%s) = $%d", gameHistoryResultExpression, len(filteredArgs)))
		}
	}
	filteredWhereSQL := ""
	if len(filteredWhereParts) > 0 {
		filteredWhereSQL = "WHERE " + strings.Join(filteredWhereParts, " AND ")
	}

	countQuery := baseCTE + `
		SELECT COUNT(*)
		FROM user_round_history item
		` + filteredWhereSQL
	if err := r.db.QueryRow(ctx, countQuery, filteredArgs...).Scan(&response.Total); err != nil {
		return response, fmt.Errorf("count user game history: %w", err)
	}

	offset := (page - 1) * pageSize
	listArgs := append(filteredArgs, pageSize, offset)
	listQuery := fmt.Sprintf(`
		%s
		SELECT
			item.rounds_id,
			item.room_id,
			item.game_id,
			item.name_game,
			item.round_status,
			%s AS result,
			item.reserved_seats,
			item.winning_seats,
			item.reserved_seats_count,
			item.winning_seats_count,
			item.entry_fee,
			item.boost_fee,
			(item.entry_fee + item.boost_fee) AS total_spent,
			item.winning_money,
			(item.winning_money - item.entry_fee - item.boost_fee) AS net_result,
			item.joined_at,
			item.finished_at
		FROM user_round_history item
		%s
		ORDER BY item.joined_at DESC, item.rounds_id DESC
		LIMIT $%d OFFSET $%d
	`, baseCTE, gameHistoryResultExpression, filteredWhereSQL, len(filteredArgs)+1, len(filteredArgs)+2)

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
	var reservedSeats []int32
	var winningSeats []int32

	err := row.Scan(
		&item.RoundID,
		&item.RoomID,
		&item.GameID,
		&item.GameName,
		&item.RoundStatus,
		&item.Result,
		&reservedSeats,
		&winningSeats,
		&item.ReservedSeatsCount,
		&item.WinningSeatsCount,
		&item.EntryFee,
		&item.BoostFee,
		&item.TotalSpent,
		&item.WinningMoney,
		&item.NetResult,
		&item.JoinedAt,
		&archivedAt,
	)
	if err != nil {
		return item, err
	}

	item.ReservedSeats = convertSeatList(reservedSeats)
	item.WinningSeats = convertSeatList(winningSeats)
	item.JoinedAt = item.JoinedAt.UTC()
	if archivedAt.Valid {
		finishedAt := archivedAt.Time.UTC()
		item.FinishedAt = &finishedAt
	}

	return item, nil
}

func convertSeatList(values []int32) []int {
	if len(values) == 0 {
		return []int{}
	}

	result := make([]int, 0, len(values))
	for _, value := range values {
		result = append(result, int(value))
	}
	return result
}

func validateGameHistoryStatus(status string) error {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "won", "lost", "left", "cancelled", "waiting", "active", "finished":
		return nil
	default:
		return fmt.Errorf("%w: unsupported game history status %q", repository.ErrorInvalidListParams, status)
	}
}
