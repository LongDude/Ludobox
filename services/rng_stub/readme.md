# RNG Stub Service

Simple stub RNG microservice for local contour.

## Endpoints

- `GET /healthz` -> service health status.
- `GET /random` -> random number.
- `GET /random?min=10&max=99` -> random number in range `[10, 99]`.

## Environment

- `HTTP_PORT` (default `8080`).
