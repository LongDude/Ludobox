# Game Server

`services/game_server` — игровой рантайм Ludobox. Он обслуживает комнаты, участников, раунды, игровые события и восстановление состояния после рестарта.

## Что делает сервис

- Поднимает HTTP API для игровых комнат и раундов.
- Регистрирует инстанс в shared PostgreSQL.
- Поддерживает heartbeat и ownership комнат.
- Хранит активное состояние комнаты в собственном Redis инстанса.
- Восстанавливает кэш и состояние после перезапуска.

## Особенности интеграции

- В общем контуре всегда работают два инстанса:
  - `game-server-core-1` -> `game-redis-1`
  - `game-server-core-2` -> `game-redis-2`
- HAProxy выполняет sticky routing по `room_id`, чтобы одна и та же комната всегда попадала в один и тот же инстанс.
- Если `room_id` отсутствует в room-scoped запросе, gateway возвращает `400`.

## Основные маршруты

- `GET /api/game/healthz`
- `GET /api/game/rooms/:roomID`
- `POST /api/game/rooms/:roomID/join`
- `POST /api/game/rooms/:roomID/join-seat`
- `POST /api/game/rooms/:roomID/leave`
- `GET /api/game/rooms/:roomID/rounds/:roundID`
- `GET /api/game/rooms/:roomID/rounds/:roundID/events`
- `POST /internal/rounds/:roundID/start`
- `POST /internal/rounds/:roundID/finalize`

Подробный контракт API:

- `docs/swagger.yaml`
- `docs/swagger.json`

## Запуск в общем контуре

Из корня:

```bash
make up
```

Ключевые переменные в корневом `.env`:

- `SHARED_POSTGRES_*`
- `GAME_REDIS_PASSWORD`
- `GAME_SERVER_HTTP_PORT`
- `GAME_SERVER_SWAGGER_BASE_PATH`
- `GAME_SERVER_HEARTBEAT_INTERVAL`
- `RNG_SERVICE_URL`

## Standalone-запуск

Для изолированной разработки доступны локальные:

- `docker-compose.yml`
- `dotenv`

Но основной интеграционный сценарий теперь описан в корневом Compose.

## Структура директории

- `cmd/` — запуск сервиса.
- `internal/service/` — игровая логика комнат, событий и таймеров.
- `internal/repository/postgres/` — shared persistence.
- `internal/repository/redis/` — session/cache слой текущего инстанса.
- `nginx/` — standalone reverse proxy для отдельного запуска.
- `docs/` — Swagger.
