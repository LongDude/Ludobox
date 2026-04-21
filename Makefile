help:
	@echo "build - start containers with precompilation"
	@echo "up - start containers"
	@echo "down - stop containers"

up:
	@docker compose up -d --scale matchmaking-core=3 --scale user-service-core=2

build:
	@docker compose up -d --build --scale matchmaking-core=3 --scale user-service-core=2

build-sso:
	@docker compose build sso-core

build-user:
	@docker compose build user-service-core

build-game:
	@docker compose build game-server-core-2 game-server-core-1

build-matchmaking:
	@docker compose build matchmaking-core

down:
	@docker compose down