# Ludobox

Ludobox — микросервисная платформа для проведения игр и лотерей. Корневой `docker-compose.yml` теперь является единой точкой входа для backend-контура, фронтенда и вспомогательных инструментов, а все общие переменные окружения задаются из корневого файла `.env`.

## Архитектура

![Architecture.png](docs/Architecture.png)

## Состав проекта

- `services/sso` — авторизация, сессии, OAuth и внутренний gRPC API.
- `services/game_server` — игровой рантайм, комнаты, раунды и sticky routing по `room_id`.
- `services/matchmaking` — рекомендации комнат и выбор игрового инстанса.
- `services/user_service` — профиль пользователя, история, баланс, игры, конфиги и административные API.
- `services/haproxy` — единый API gateway и балансировщик.
- `services/rng_stub` — локальная заглушка RNG для сценариев разработки.
- `apps/frontend` — клиентское приложение на Vue 3.
- `tools/pgadmin` и `tools/redis-commander` — вспомогательные UI-инструменты.
- `db/migrations` — общие миграции для shared PostgreSQL.
- `docs` — индекс документации и ссылки на Swagger/схемы.

## Карта документации

- [Корневая документация](README.md)
- [Справочник по документации](docs/README.md)
- [Frontend](apps/frontend/README.md)
- [SSO](services/sso/readme.md)
- [Game Server](services/game_server/readme.md)
- [Matchmaking](services/matchmaking/readme.md)
- [User Service](services/user_service/readme.md)
- [pgAdmin](tools/pgadmin/README.md)
- [Redis Commander](tools/redis-commander/README.md)

## Централизованная конфигурация

- Основной источник переменных окружения для общего запуска: корневой `.env`.
- Корневой `docker-compose.yml` больше не опирается на локальные `.env` сервисов.
- Для shared и SSO окружений используются отдельные префиксованные переменные в `.env`, а Compose прокладывает их в ожидаемые имена внутри контейнеров.
- Локальные файлы `dotenv` внутри сервисов оставлены для изолированного standalone-запуска конкретного сервиса из его директории.

Ключевые группы переменных в корневом `.env`:

- Общие: `DOMAIN`, `PUBLIC_URL`, `ALLOWED_*`, `DEFAULT_ADMIN_EMAILS`, `INTERNAL_PROXY_TOKEN`.
- Shared storage: `SHARED_POSTGRES_*`, `*_REDIS_PASSWORD`, `*_REDIS_DB`.
- SSO: `SSO_POSTGRES_*`, `SSO_HTTP_PORT`, `SSO_GRPC_*`, OAuth/SMTP/JWT параметры.
- Backend API: `GAME_SERVER_*`, `MATCHMAKING_*`, `USER_SERVICE_*`, `RNG_*`.
- Frontend: `FRONTEND_DEV_PORT`, `FRONTEND_PORT`, `FRONTEND_*_BASE_URL`, `VITE_API_BASE_URL`, `VITE_SSO_CLIENT_ID_URL`.
- Tools: `PGADMIN_*`, `REDIS_COMMANDER_*_PORT`.

## Профили Compose

Сервисы backend-контура запускаются по умолчанию, без профилей.

Дополнительные профили:

- `frontend-dev` — dev-сервер фронтенда.
- `frontend-prod` — production-сборка фронтенда.
- `tools-pgadmin` — pgAdmin.
- `tools-redis` — Redis Commander для игровых и служебных Redis.

## Быстрый старт

1. Проверьте и заполните корневой `.env`.
2. Поднимите основной backend-контур:

```bash
make up
```

3. При необходимости поднимите фронтенд:

```bash
make frontend-up
```

4. При необходимости поднимите инструменты:

```bash
make tools-up
```

5. Откройте:

- API gateway: `http://localhost`
- HAProxy stats: `http://localhost:8404/stats`
- Frontend dev: `http://localhost:5173`
- Frontend prod: `http://localhost:8080`
- pgAdmin: `http://localhost:8081`
- Redis Commander: `http://localhost:9001`, `9002`, `9003`, `9004`

## Основные команды Makefile

```bash
make help
make up
make build
make down
make frontend-up
make frontend-up-prod
make frontend-down
make tools-up
make tools-down
make pgadmin-up
make redis-up
```

По умолчанию корневой `Makefile` поднимает:

- `matchmaking-core` с масштабом `3`
- `user-service-core` с масштабом `2`

Масштаб можно переопределить на запуске:

```bash
make MATCHMAKING_SCALE=2 USER_SERVICE_SCALE=1 up
```

## Сетевая схема

- Весь стек работает в сети `ludobox_rooms_network`.
- `haproxy` маршрутизирует внешние запросы на backend-сервисы.
- `game_server` работает в двух фиксированных инстансах, каждый со своим Redis.
- Sticky routing для комнат завязан на `room_id`, который HAProxy извлекает из заголовка, query string или path.

## Standalone-режим сервисов

Если нужен запуск одного сервиса в отрыве от общего стека:

- переходите в директорию сервиса;
- используйте локальный `docker-compose.yml` и `dotenv` этого сервиса;
- ориентируйтесь на README внутри соответствующей директории.

Для общего локального окружения корневой Compose остаётся основным сценарием.
