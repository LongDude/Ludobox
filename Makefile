COMPOSE ?= docker compose
MATCHMAKING_SCALE ?= 3
USER_SERVICE_SCALE ?= 2
BACKEND_SCALE = --scale matchmaking-core=$(MATCHMAKING_SCALE) --scale user-service-core=$(USER_SERVICE_SCALE)
TOOLS_SERVICES = pgadmin redis-commander-game-1 redis-commander-game-2 redis-commander-matchmaking redis-commander-user
FRONTEND_SERVICES = frontend-dev frontend-prod

.PHONY: help up build down build-sso build-user build-game build-matchmaking build-rng \
	frontend-up frontend-dev-up frontend-prod-up frontend-down frontend-dev-down frontend-prod-down \
	tools-up tools-down pgadmin-up pgadmin-down redis-up redis-down

help:
	@echo "-- Запуск серверной части --"
	@echo "up                 - стартовать основной backend-контур"
	@echo "build              - пересобрать и стартовать основной backend-контур"
	@echo "down               - остановить весь проект"
	@echo "build-sso          - пересобрать образ sso-core"
	@echo "build-user         - пересобрать образ user-service-core"
	@echo "build-game         - пересобрать образы game-server-core-*"
	@echo "build-matchmaking  - пересобрать образ matchmaking-core"
	@echo "build-rng          - пересобрать образ rng-stub"
	@echo ""
	@echo "-- Запуск клиентской части --"
	@echo "frontend-up        - поднять frontend в dev-профиле"
	@echo "frontend-dev-up    - поднять frontend-dev"
	@echo "frontend-prod-up   - поднять frontend-prod"
	@echo "frontend-down      - остановить оба frontend-профиля"
	@echo ""
	@echo "-- Вспомогательные утилиты для отладки"
	@echo "tools-up           - поднять pgadmin и redis-commander"
	@echo "tools-down         - остановить все tool-сервисы"
	@echo "pgadmin-up         - поднять только pgadmin"
	@echo "pgadmin-down       - остановить только pgadmin"
	@echo "redis-up           - поднять все redis-commander"
	@echo "redis-down         - остановить все redis-commander"

# === Серверная часть ===
up:
	@$(COMPOSE) up -d $(BACKEND_SCALE)

build:
	@$(COMPOSE) up -d --build $(BACKEND_SCALE)

build-sso:
	@$(COMPOSE) build sso-core

build-user:
	@$(COMPOSE) build user-service-core

build-game:
	@$(COMPOSE) build game-server-core-1 game-server-core-2

build-matchmaking:
	@$(COMPOSE) build matchmaking-core

build-rng:
	@$(COMPOSE) build rng-stub

# === Клиентская часть ===
frontend-up: frontend-dev-up

frontend-dev-up:
	@$(COMPOSE) --profile frontend-dev up -d frontend-dev

frontend-prod-up:
	@$(COMPOSE) --profile frontend-prod up -d frontend-prod

frontend-down:
	-@$(COMPOSE) stop $(FRONTEND_SERVICES)
	-@$(COMPOSE) rm -f $(FRONTEND_SERVICES)

frontend-dev-down:
	-@$(COMPOSE) stop frontend-dev
	-@$(COMPOSE) rm -f frontend-dev

frontend-prod-down:
	-@$(COMPOSE) stop frontend-prod
	-@$(COMPOSE) rm -f frontend-prod

# Вспомогательные приложения
tools-up:
	@$(COMPOSE) --profile tools-pgadmin --profile tools-redis up -d $(TOOLS_SERVICES)

tools-down:
	-@$(COMPOSE) stop $(TOOLS_SERVICES)
	-@$(COMPOSE) rm -f $(TOOLS_SERVICES)

pgadmin-up:
	@$(COMPOSE) --profile tools-pgadmin up -d pgadmin

pgadmin-down:
	-@$(COMPOSE) stop pgadmin
	-@$(COMPOSE) rm -f pgadmin

redis-up:
	@$(COMPOSE) --profile tools-redis up -d redis-commander-game-1 redis-commander-game-2 redis-commander-matchmaking redis-commander-user

redis-down:
	-@$(COMPOSE) stop redis-commander-game-1 redis-commander-game-2 redis-commander-matchmaking redis-commander-user
	-@$(COMPOSE) rm -f redis-commander-game-1 redis-commander-game-2 redis-commander-matchmaking redis-commander-user

# Выключение сервера
down:
	@$(COMPOSE) down
