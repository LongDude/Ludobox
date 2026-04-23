# Frontend

`apps/frontend` — клиентское приложение Ludobox на Vue 3, Vite и TypeScript. В общем локальном контуре фронтенд подключён к корневому `docker-compose.yml` через отдельные профили и использует переменные `VITE_*` из корневого `.env`.

## Стек

- Vue 3
- TypeScript
- Vite
- Pinia
- Vue Router
- Axios
- Vitest

## Профили в корневом Compose

- `frontend-dev` — dev-сервер с hot reload.
- `frontend-prod` — production-сборка и раздача через nginx.

Команды из корня проекта:

```bash
make frontend-up
make frontend-dev-up
make frontend-prod-up
make frontend-prod-cert
make frontend-prod-renew
make frontend-vps-up
make frontend-down
```

Порты задаются в корневом `.env`:

- `FRONTEND_DEV_PORT`
- `FRONTEND_PORT`

## Переменные окружения

Для общего запуска фронтенда используются значения из корневого `.env`:

- `VITE_API_BASE_URL`
- `VITE_SSO_CLIENT_ID_URL`
- `FRONTEND_DEV_BASE_URL`
- `FRONTEND_BASE_URL`

Эти переменные пробрасываются:

- в `frontend-dev` как `VITE_FRONTEND_BASE_URL=${FRONTEND_DEV_BASE_URL}`;
- в `frontend-prod` как `VITE_FRONTEND_BASE_URL=${FRONTEND_BASE_URL}` на стадии сборки;
- `VITE_API_BASE_URL` и `VITE_SSO_CLIENT_ID_URL` используются в обоих профилях одинаково.

## Локальный запуск без корневого Compose

Из директории `apps/frontend`:

```bash
npm install
npm run dev
```

Для standalone-запуска можно продолжать использовать локальный `dotenv` или собственный `.env.local`.

## Основные директории

- `src/views/` — страницы приложения.
- `src/components/` — UI-компоненты.
- `src/stores/` — Pinia stores.
- `src/api/` — интеграция с backend API.
- `src/assets/` — базовые стили и статические ассеты.

## Проверка качества

```bash
npm run test:unit
npm run lint
npm run build
```

## Production TLS

- Локальный `apps/frontend/docker-compose.yml` остаётся standalone-сценарием только для изолированной разработки фронтенда.
- VPS и production TLS теперь поднимаются через корневые `docker-compose.yml`, `Makefile` и общий `haproxy`.
- Для генерации и продления сертификатов используется сервис `frontend-certbot`.
- Сертификаты и ACME webroot сохраняются в `apps/frontend/certbot/conf` и `apps/frontend/certbot/www`.
- Перед `make frontend-vps-up` или `make frontend-prod-cert` заполните в корневом `.env`:
  - `DOMAIN`
  - `LETSENCRYPT_EMAIL`
  - `PUBLIC_URL=https://DOMAIN`
  - `FRONTEND_BASE_URL=https://DOMAIN`
  - `VITE_API_BASE_URL=https://DOMAIN/api`
  - `ALLOWED_REDIRECT_URLS` и `ALLOWED_CORS_ORIGINS`, включив `https://DOMAIN`
- Команды для production-сценария:
  - `make frontend-prod-up` — поднять `frontend-prod` и `haproxy` без выпуска сертификата.
  - `make frontend-prod-cert` — выпустить или переиспользовать сертификат Let's Encrypt.
  - `make frontend-prod-renew` — продлить сертификаты.
  - `make frontend-vps-up` — полный запуск backend + production frontend + выпуск сертификата.
