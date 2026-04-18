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
docker compose up --build -d --scale matchmaking-core=3 --scale user-service-core=2 --scale game-server-core=2