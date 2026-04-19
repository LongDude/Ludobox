package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"user_service/internal/config"
	"user_service/internal/domain"
	"user_service/internal/repository"

	"github.com/sirupsen/logrus"
)

type RoomService interface {
	CreateRoomByConfigID(ctx context.Context, configID int) (*domain.Room, error)
	GetRoomByID(ctx context.Context, id int) (*domain.Room, error)
	GetNotArchivedRooms(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Room], error)
	UpdateRoomByID(ctx context.Context, id int, room *domain.Room) (*domain.Room, error)
	DeleteRoomByID(ctx context.Context, id int) error
}

type roomService struct {
	roomRepository             repository.RoomRepository
	configRepository           repository.ConfigRepository
	matchmakingSelectServerURL string
	internalProxyToken         string
	httpClient                 *http.Client
	logger                     *logrus.Logger
}

type matchmakingServerResponse struct {
	ServerID int `json:"server_id"`
}

type matchmakingErrorResponse struct {
	Error string `json:"error"`
}

func NewRoomService(roomRepository repository.RoomRepository, configRepository repository.ConfigRepository, cfg *config.Config, logger *logrus.Logger) RoomService {
	return &roomService{
		roomRepository:             roomRepository,
		configRepository:           configRepository,
		matchmakingSelectServerURL: cfg.MatchmakingSelectServerURL,
		internalProxyToken:         cfg.InternalProxyToken,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

func (r *roomService) CreateRoomByConfigID(ctx context.Context, configID int) (*domain.Room, error) {
	if configID <= 0 {
		return nil, fmt.Errorf("%w: config_id must be positive", repository.ErrorInvalidRoom)
	}

	config, err := r.configRepository.GetConfigByID(ctx, configID)
	if err != nil {
		return nil, err
	}
	if config.ArchivedAt != nil {
		return nil, repository.ErrorConfigArchived
	}

	serverID, err := r.selectAvailableGameServer(ctx)
	if err != nil {
		return nil, err
	}

	return r.roomRepository.CreateRoomByConfigID(ctx, configID, serverID)
}

func (r *roomService) GetRoomByID(ctx context.Context, id int) (*domain.Room, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: room_id must be positive", repository.ErrorInvalidRoom)
	}

	return r.roomRepository.GetRoomByID(ctx, id)
}

func (r *roomService) GetNotArchivedRooms(ctx context.Context, params domain.ListParams) (domain.ListResponse[domain.Room], error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	if params.Sort == nil {
		params.Sort = &domain.Sort{
			Field:     "room_id",
			Direction: "desc",
		}
	} else {
		params.Sort.Field = strings.TrimSpace(params.Sort.Field)
		params.Sort.Direction = strings.ToLower(strings.TrimSpace(params.Sort.Direction))
		if params.Sort.Field == "" {
			return domain.ListResponse[domain.Room]{}, fmt.Errorf("%w: sort field cannot be empty", repository.ErrorInvalidListParams)
		}
		if params.Sort.Direction == "" {
			params.Sort.Direction = "asc"
		}
	}

	return r.roomRepository.GetNotArchivedRooms(ctx, params)
}

func (r *roomService) UpdateRoomByID(ctx context.Context, id int, room *domain.Room) (*domain.Room, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: room_id must be positive", repository.ErrorInvalidRoom)
	}
	if room == nil {
		return nil, fmt.Errorf("%w: request body is required", repository.ErrorInvalidRoom)
	}

	current, err := r.roomRepository.GetRoomByID(ctx, id)
	if err != nil {
		return nil, err
	}

	current.GameServerID = room.GameServerID
	current.ArchivedAt = room.ArchivedAt

	return r.roomRepository.UpdateRoomByID(ctx, id, current)
}

func (r *roomService) DeleteRoomByID(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("%w: room_id must be positive", repository.ErrorInvalidRoom)
	}

	return r.roomRepository.DeleteRoomByID(ctx, id)
}

func (r *roomService) selectAvailableGameServer(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.matchmakingSelectServerURL, nil)
	if err != nil {
		return 0, fmt.Errorf("build matchmaking request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Internal-Proxy-Token", r.internalProxyToken)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request matchmaking server selection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		var payload matchmakingErrorResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr == nil && payload.Error != "" {
			return 0, fmt.Errorf("%w: %s", repository.ErrorNoActiveGameServers, payload.Error)
		}
		return 0, repository.ErrorNoActiveGameServers
	}
	if resp.StatusCode != http.StatusOK {
		var payload matchmakingErrorResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr == nil && payload.Error != "" {
			return 0, fmt.Errorf("matchmaking server selection failed with status %d: %s", resp.StatusCode, payload.Error)
		}
		return 0, fmt.Errorf("matchmaking server selection failed with status %d", resp.StatusCode)
	}

	var payload matchmakingServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("decode matchmaking server response: %w", err)
	}
	if payload.ServerID <= 0 {
		return 0, fmt.Errorf("matchmaking returned invalid server_id")
	}

	return payload.ServerID, nil
}
