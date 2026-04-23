# k6 Load Tests

Корневой каталог для нагрузочных сценариев k6.

## Сценарии

- `matchmaking.js` — `healthz`, `rooms/recommendations`, `rooms/quick-match`
- `game_server.js` — `healthz`, `rooms/:roomID`, `rounds/:roundID`, optional cleanup через `leave`

По умолчанию сценарии идут через gateway `http://localhost`, логинятся в SSO по `LOGIN` и `PASSWORD`, а затем используют полученный `Authorization: Bearer`.

Если не нужно нагружать логин, можно передать готовый `ACCESS_TOKEN`.

## Переменные окружения

- `BASE_URL` — адрес gateway, по умолчанию `http://localhost`
- `LOGIN`, `PASSWORD` — учётные данные для входа в SSO
- `ACCESS_TOKEN` — готовый access token вместо логина
- `STRICT=1` — падать на неуспешных проверках
- `SLEEP_MS` — пауза между итерациями
- `GAME_ID`, `MIN_REGISTRATION_PRICE`, `MAX_REGISTRATION_PRICE`, `MIN_CAPACITY`, `MAX_CAPACITY`, `IS_BOOST`, `MIN_BOOST_POWER` — фильтры matchmaking
- `PAGE`, `PAGE_SIZE` — пагинация для `recommendations`
- `ENABLE_QUICK_MATCH=0` — отключить `quick-match` в `matchmaking.js`
- `ROOM_ID`, `ROUND_ID` — зафиксировать контекст для `game_server.js` без `quick-match`
- `LEAVE_AFTER=0` — не выполнять cleanup `POST /rooms/:roomID/leave` после `quick-match`

## Запуск через Makefile

```bash
make k6-matchmaking K6_LOGIN=user@example.com K6_PASSWORD=secret
make k6-game-server K6_LOGIN=user@example.com K6_PASSWORD=secret
make k6-game-server K6_ACCESS_TOKEN=... K6_ROOM_ID=101 K6_ROUND_ID=5001
```

Дополнительные флаги k6 можно пробрасывать через `K6_ARGS`, например:

```bash
make k6-matchmaking K6_LOGIN=user@example.com K6_PASSWORD=secret K6_ARGS="--vus 20 --duration 1m"
```
