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
make frontend-prod-up
make frontend-prod-cert
make frontend-prod-renew
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

- The standalone `apps/frontend/docker-compose.yml` remains a local-only flow.
- VPS deployment uses the root `docker-compose.yml`, root `Makefile`, and the root `haproxy`.
- Before `make frontend-vps-up`, set these values in the root `.env`:
  - `DOMAIN`
  - `LETSENCRYPT_EMAIL`
  - `PUBLIC_URL=https://DOMAIN`
  - `FRONTEND_BASE_URL=https://DOMAIN`
  - `VITE_API_BASE_URL=https://DOMAIN/api`
