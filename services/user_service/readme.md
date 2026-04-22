# User Service

`services/user_service` обслуживает пользовательский профиль, игровую историю, баланс, рейтинговую историю, каталог игр, конфиги, комнаты и административные сценарии платформы.

## Что делает сервис

- CRUD по пользователю и его балансу.
- История игр и рейтинга.
- Административное управление играми, конфигами, комнатами и серверами.
- SSE-потоки для admin events и user balance events.
- Интеграция с SSO и Matchmaking через внутренние HTTP вызовы.

## Основные маршруты

Публичные:

- `GET /api/users/healthz`
- `GET /api/users/user`
- `POST /api/users/user`
- `PUT /api/users/user`
- `DELETE /api/users/user`
- `PUT /api/users/user/balance`
- `GET /api/users/user/balance/events`
- `GET /api/users/user/history/games`
- `GET /api/users/user/history/rating`

Административные:

- `GET /api/users/games`
- `POST /api/users/game`
- `GET /api/users/configs/used`
- `POST /api/users/config`
- `GET /api/users/rooms`
- `POST /api/users/room`
- `GET /api/users/servers`
- `GET /api/users/events`

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
make USER_SERVICE_SCALE=2 up
```

Ключевые переменные в корневом `.env`:

- `SHARED_POSTGRES_*`
- `USER_REDIS_PASSWORD`
- `USER_SERVICE_HTTP_PORT`
- `USER_SERVICE_SWAGGER_BASE_PATH`
- `SSO_AUTHENTICATE_URL`
- `MATCHMAKING_SELECT_SERVER_URL`
- `INTERNAL_PROXY_TOKEN`

## Standalone-запуск

Для отдельной разработки внутри директории можно использовать:

- `docker-compose.yml`
- `dotenv`

Но общий сценарий для платформы описан в корневом Compose.

## Структура директории

- `cmd/` — entrypoint.
- `internal/service/` — бизнес-логика пользователей, игр, конфигов и событий.
- `internal/repository/postgres/` — доступ к shared БД.
- `internal/transport/http/` — роутинг, handlers, presenters, middleware.
- `nginx/` — standalone reverse proxy.
- `docs/` — Swagger.
