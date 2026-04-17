# Ludobox

## Компоненты
### Backend / Infrastructure
- `services/gateway` — точка входа API, интеграция сервисов и внешних систем.
### Frontend
- `apps/frontend` — веб-интерфейс (Vue 3), роли пользователей, кабинет, панели админа/модератора.
## Быстрый старт (локально)
> [!TIP]
> В репозитории есть отдельные `docker-compose.yml` внутри сервисов. Для запуска конкретного компонента перейдите в его каталог и следуйте README.
- [Frontend](apps/frontend/README.md)
- [Gateway](services/gateway/readme.md)