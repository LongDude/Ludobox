help:
	@echo "up-build - start containers with precompilation"
	@echo "up - start containers"
	@echo "down - stop containers"

up:
	@docker compose up -d --scale matchmaking-core=3 --scale user-service-core=2

up-build:
	@docker compose up -d --build --scale matchmaking-core=3 --scale user-service-core=2

down:
	@docker compose down