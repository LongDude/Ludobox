# Ludobox
Web-Сервис для проведения лотерей построенный на микросервисной архитектуре.
## Компоненты
### Backend / Infrastructure
- `services/haproxy` — точка входа API, интеграция сервисов и внешних систем.
- `services/game_server` — сервис отвечающий за проведение игр в комнатах.
- `services/matchmaking` — сервис отвечающий за подбор игр пользователям.
- `services/user_service` — сервис отвечающий за общее управление данными пользователя и администрирования системы.
- `services/sso` — сервис отвечающий за авторизацию и идентификацию пользователей, управляет пользовательскими сессиями.
### Frontend
- `apps/frontend` — веб-интерфейс (Vue 3), роли пользователей, кабинет, панели админа/модератора.
## Быстрый старт

## Быстрый старт (локально распределенно)
> [!TIP]
> В репозитории есть отдельные `docker-compose.yml` внутри сервисов. Для запуска конкретного компонента перейдите в его каталог и следуйте README.
> - [Frontend](apps/frontend/README.md)
> - [SSO](services/sso/readme.md)
> - [Game Server](services/game_server/readme.md)
> - [Matchmaking Server](services/matchmaking/readme.md)
> - [User Server](services/user_service/readme.md)

Для создания нескольких инстантов
docker compose up --build -d --scale matchmaking-core=3 --scale user-service-core=2

### Sticky routing для `game_server`
- `haproxy` маршрутизирует room-scoped запросы в `game_server` по `room_id`, чтобы одна и та же комната всегда попадала в один и тот же инстанс `game_server`.
- `game_server` поднимается как две явные пары:
  - `game-server-core-1` -> `game-redis-1`
  - `game-server-core-2` -> `game-redis-2`
- Это важно, потому что у каждого `game_server` свой Redis и активное состояние комнаты не должно "скакать" между инстансами.
- Источники `room_id`, которые понимает gateway:
  - заголовок `X-Room-ID`
  - query-параметр `room_id`
  - path вида `/api/game/rooms/<room_id>` или `/api/game/rounds/<room_id>`
- Для room-scoped create/update/delete запросов `room_id` обязателен уже на входе в gateway. Иначе `haproxy` вернёт `400`, чтобы комната не создалась на случайном инстансе и не потерялась для последующих sticky-запросов.
- Если `room_id` не передан, запрос уходит в обычный `roundrobin` backend `game_server`.
