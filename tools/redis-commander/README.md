# Redis Commander

`tools/redis-commander` — набор UI-клиентов для Redis инстансов Ludobox. В общем локальном окружении эти сервисы уже включены в корневой `docker-compose.yml` и вынесены в профиль `tools-redis`.

## Что поднимается

- `redis-commander-game-1` -> `http://localhost:9001`
- `redis-commander-game-2` -> `http://localhost:9002`
- `redis-commander-matchmaking` -> `http://localhost:9003`
- `redis-commander-user` -> `http://localhost:9004`

## Запуск из корня

```bash
make redis-up
make redis-down
```

Или вместе со всеми инструментами:

```bash
make tools-up
make tools-down
```

Параметры берутся из корневого `.env`:

- `REDIS_COMMANDER_GAME_1_PORT`
- `REDIS_COMMANDER_GAME_2_PORT`
- `REDIS_COMMANDER_MATCHMAKING_PORT`
- `REDIS_COMMANDER_USER_PORT`
- `GAME_REDIS_PASSWORD`
- `MATCHMAKING_REDIS_PASSWORD`
- `USER_REDIS_PASSWORD`

## Standalone-режим

Локальный `docker-compose.yml` в этой директории сохранён для изолированного запуска, но для общей локальной среды рекомендуем использовать корневой Compose.
