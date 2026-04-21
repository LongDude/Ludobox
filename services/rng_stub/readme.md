# RNG Stub Service

Simple stub RNG microservice for local contour.

## Endpoints

- `GET /healthz` -> service health status.
- `GET /random` -> random number.
- `GET /random?min=10&max=99` -> random number in range `[10, 99]`.
- `POST /winnings/distribute` -> converts an array of probabilities into `m` winning positions.

Request example:

```json
{
  "probabilities": [0.1, 0.3, 0.6],
  "winners_count": 2
}
```

Response example:

```json
{
  "winning_positions": [3, 2],
  "winners_count": 2,
  "positions_count": 3
}
```

Notes:
- positions in `winning_positions` are 1-based.
- selection is weighted and without replacement (no duplicate positions in one draw).

## Environment

- `HTTP_PORT` (default `8080`).
