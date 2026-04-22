# Matchmaking Service

`services/matchmaking` отвечает за рекомендации комнат, quick-match сценарии и выдачу подходящего игрового сервера для создания или продолжения игры.

## Что делает сервис

- Отдаёт публичные рекомендации по комнатам.
- Поддерживает быстрый подбор комнаты.
- Через internal API выбирает доступный `game_server`.
- Использует shared PostgreSQL и отдельный Redis для служебного кэша.

## Основные маршруты

- `GET /api/matchmaking/healthz`
- `GET /api/matchmaking/rooms/recommendations`
- `GET /api/matchmaking/rooms/quick-match`
- `GET /__internal/matchmaking/rooms/:room_id/owner`
- `GET /__internal/matchmaking/game-servers/next`

Подробный контракт API:

- `docs/swagger.yaml`
- `docs/swagger.json`

## Запуск в общем контуре

Из корня:

```bash
make up
```

По умолчанию корневой `Makefile` масштабирует сервис:

```bash
make MATCHMAKING_SCALE=3 up
```

Ключевые переменные в корневом `.env`:

- `SHARED_POSTGRES_*`
- `MATCHMAKING_REDIS_PASSWORD`
- `MATCHMAKING_HTTP_PORT`
- `MATCHMAKING_SWAGGER_BASE_PATH`
- `GAME_SERVER_STALE_AFTER`
- `MATCHMAKING_RECOMMENDATION_CACHE_TTL`

## Standalone-запуск

Для отдельной разработки внутри директории можно использовать:

- `docker-compose.yml`
- `dotenv`

Для интеграционного запуска платформы используйте корневой Compose.

## Структура директории

- `cmd/` — entrypoint.
- `internal/service/` — подбор комнат и серверов.
- `internal/repository/postgres/` — чтение shared состояния.
- `internal/repository/redis/` — кэш/служебные данные.
- `nginx/` — standalone reverse proxy.
- `docs/` — Swagger.
