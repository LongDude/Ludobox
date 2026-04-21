help:
	@echo "up - start containers"
	@echo "build - start containers with precompilation"
	@echo "build-sso - "
	@echo "build-user - "
	@echo "build-game - "
	@echo "build-matchmaking - "
	@echo "build-rng - "
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

build-rng:
	@docker compose build rng-stub

down:
	@docker compose down