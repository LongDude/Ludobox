# pgAdmin

`tools/pgadmin` — вспомогательный UI для ручной работы с PostgreSQL. В основном локальном окружении сервис уже включён в корневой `docker-compose.yml` и вынесен в профиль `tools-pgadmin`.

## Запуск из корня

```bash
make pgadmin-up
make pgadmin-down
```

Или:

```bash
make tools-up
make tools-down
```

После запуска интерфейс доступен по адресу:

- `http://localhost:8081`

Параметры берутся из корневого `.env`:

- `PGADMIN_PORT`
- `PGADMIN_DEFAULT_EMAIL`
- `PGADMIN_DEFAULT_PASSWORD`

## Подключения внутри pgAdmin

Основные хосты внутри docker-сети:

- `shared-postgres`
- `sso-postgres`

Порты внутри сети у обоих сервисов стандартные: `5432`.

## Standalone-режим

Локальный `docker-compose.yml` в этой директории сохранён для изолированного запуска, но основной сценарий теперь проходит через корневой Compose.
