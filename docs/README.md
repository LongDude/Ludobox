# Документация Ludobox

Эта директория хранит проектный справочник и ссылки на артефакты, которые не стоит дублировать в README сервисов.

## Что где лежит

- `../README.md` — корневая карта проекта и общий сценарий запуска.
- `../db/migrations` — общие SQL-миграции shared PostgreSQL.
- `../services/sso/docs/swagger.yaml` — Swagger SSO.
- `../services/game_server/docs/swagger.yaml` — Swagger Game Server.
- `../services/matchmaking/docs/swagger.yaml` — Swagger Matchmaking.
- `../services/user_service/docs/swagger.yaml` — Swagger User Service.

## Как читать документацию

1. Начинайте с [корневого README](../README.md).
2. Для конкретного компонента переходите в его README в `services/`, `apps/` или `tools/`.
3. Для точных контрактов API используйте соответствующий Swagger из директории `docs` сервиса.

## Примечания

- Корневой `docker-compose.yml` и корневой `.env` — основной источник правды для локального интеграционного окружения.
- Локальные `docker-compose.yml` внутри сервисов и инструментов нужны в основном для изолированной разработки конкретного компонента.
