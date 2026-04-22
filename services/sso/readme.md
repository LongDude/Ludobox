# SSO Service

`services/sso` отвечает за регистрацию, логин, refresh/logout, OAuth-вход и внутреннюю gRPC-интеграцию. В общем локальном контуре сервис запускается из корневого `docker-compose.yml`, а переменные для него берутся из корневого `.env`.

## Ответственность сервиса

- HTTP API для auth-flow и управления пользователями.
- Redis-backed сессии и blocklist refresh/access токенов.
- OAuth-интеграции Google и Yandex.
- Внутренний gRPC API для сервисов платформы.
- Отдельная PostgreSQL и отдельный Redis.

## Основные маршруты

- `/api/auth/*` — регистрация, логин, refresh, logout, authenticate, password reset.
- `/api/oauth/*` — OAuth redirect/callback.
- `/api/auth/admin/*` — административные операции над пользователями.
- `/swagger/*` — Swagger UI и описание API.

Подробный контракт лежит в:

- `docs/swagger.yaml`
- `docs/swagger.json`

## Запуск в общем контуре

Из корня репозитория:

```bash
make up
```

Сервис зависит от:

- `sso-postgres`
- `sso-redis`
- `sso-migrator`

Ключевые переменные в корневом `.env`:

- `SSO_POSTGRES_*`
- `SSO_REDIS_*`
- `SSO_HTTP_PORT`
- `SSO_GRPC_PORT`
- `JWT_SECRET_KEY`
- `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`
- `SMTP_*`, `FROM_EMAIL`, `SMTP_JWT_SECRET`
- `GOOGLE_CLIENT_*`, `YANDEX_CLIENT_*`, `VK_CLIENT_*`

## Standalone-запуск

Для отдельной разработки SSO можно перейти в `services/sso` и использовать локальные:

- `docker-compose.yml`
- `dotenv`

Это отдельный сценарий. Для интеграционного запуска платформы используйте корневой Compose.

## Структура директории

- `cmd/` — entrypoint сервиса.
- `internal/` — конфиг, доменная логика, репозитории, HTTP/gRPC transport.
- `api/live_id/v1/` — protobuf и сгенерированные gRPC stub-файлы.
- `db/migrations/` — миграции схемы SSO.
- `tools/migrator/` — мигратор для standalone-контура SSO.
- `docs/` — Swagger.
