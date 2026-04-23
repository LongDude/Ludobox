COMPOSE ?= docker compose
MATCHMAKING_SCALE ?= 3
USER_SERVICE_SCALE ?= 2
BACKEND_SCALE = --scale matchmaking-core=$(MATCHMAKING_SCALE) --scale user-service-core=$(USER_SERVICE_SCALE)
TOOLS_SERVICES = pgadmin redis-commander-game-1 redis-commander-game-2 redis-commander-matchmaking redis-commander-user
FRONTEND_SERVICES = frontend-dev frontend-prod
FRONTEND_PROD_STACK = haproxy frontend-prod
K6 ?= k6
K6_DIR ?= tests/k6
K6_BASE_URL ?= http://localhost
K6_LOGIN ?=
K6_PASSWORD ?=
K6_ACCESS_TOKEN ?=
K6_STRICT ?= 0
K6_SLEEP_MS ?= 100
K6_GAME_ID ?=
K6_MIN_REGISTRATION_PRICE ?=
K6_MAX_REGISTRATION_PRICE ?=
K6_MIN_CAPACITY ?=
K6_MAX_CAPACITY ?=
K6_IS_BOOST ?=
K6_MIN_BOOST_POWER ?=
K6_PAGE ?= 1
K6_PAGE_SIZE ?= 10
K6_ENABLE_QUICK_MATCH ?= 1
K6_ROOM_ID ?=
K6_ROUND_ID ?=
K6_LEAVE_AFTER ?= 1
K6_ARGS ?=

.PHONY: help up build down build-sso build-user build-game build-matchmaking build-rng \
	frontend-up frontend-dev-up frontend-prod-up frontend-prod-cert frontend-prod-renew frontend-vps-up \
	frontend-down frontend-dev-down frontend-prod-down \
	tools-up tools-down pgadmin-up pgadmin-down redis-up redis-down \
	k6-matchmaking k6-game-server k6-all

help:
	@echo "-- Backend --"
	@echo "up                 - start the main backend stack"
	@echo "build              - rebuild and start the main backend stack"
	@echo "down               - stop the whole project"
	@echo "build-sso          - rebuild sso-core image"
	@echo "build-user         - rebuild user-service-core image"
	@echo "build-game         - rebuild game-server-core-* images"
	@echo "build-matchmaking  - rebuild matchmaking-core image"
	@echo "build-rng          - rebuild rng-stub image"
	@echo ""
	@echo "-- Frontend --"
	@echo "frontend-up        - start frontend in dev profile"
	@echo "frontend-dev-up    - start frontend-dev"
	@echo "frontend-prod-up   - start frontend-prod with the root haproxy ingress"
	@echo "frontend-prod-cert - issue or reuse Let's Encrypt cert for DOMAIN and restart haproxy"
	@echo "frontend-prod-renew - renew Let's Encrypt certs and restart haproxy"
	@echo "frontend-vps-up    - build backend + frontend-prod and issue cert for VPS deployment"
	@echo "frontend-down      - stop both frontend profiles"
	@echo ""
	@echo "-- Tools --"
	@echo "tools-up           - start pgadmin and redis-commander"
	@echo "tools-down         - stop all tool services"
	@echo "pgadmin-up         - start pgadmin only"
	@echo "pgadmin-down       - stop pgadmin only"
	@echo "redis-up           - start all redis-commander services"
	@echo "redis-down         - stop all redis-commander services"
	@echo ""
	@echo "-- k6 load testing --"
	@echo "k6-matchmaking     - run tests/k6/matchmaking.js"
	@echo "k6-game-server     - run tests/k6/game_server.js"
	@echo "k6-all             - run both k6 scenarios sequentially"

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

frontend-up: frontend-dev-up

frontend-dev-up:
	@$(COMPOSE) --profile frontend-dev up -d frontend-dev

frontend-prod-up:
	@$(COMPOSE) --profile frontend-prod up -d $(FRONTEND_PROD_STACK)

frontend-prod-cert:
	@$(COMPOSE) --profile frontend-prod up -d $(FRONTEND_PROD_STACK)
	@$(COMPOSE) --profile frontend-prod run --rm --entrypoint sh frontend-certbot -ec 'test -n "$$LETSENCRYPT_EMAIL" || { echo "LETSENCRYPT_EMAIL is required in .env"; exit 1; }; test "$$DOMAIN" != "localhost" || { echo "DOMAIN must be set to a public hostname in .env"; exit 1; }; certbot certonly --webroot --webroot-path /var/www/certbot --email "$$LETSENCRYPT_EMAIL" --agree-tos --no-eff-email --non-interactive --keep-until-expiring -d "$$DOMAIN"'
	@$(COMPOSE) restart haproxy

frontend-prod-renew:
	@$(COMPOSE) --profile frontend-prod up -d $(FRONTEND_PROD_STACK)
	@$(COMPOSE) --profile frontend-prod run --rm frontend-certbot renew
	@$(COMPOSE) restart haproxy

frontend-vps-up:
	@$(COMPOSE) --profile frontend-prod up -d --build $(BACKEND_SCALE) $(FRONTEND_PROD_STACK)
	@$(MAKE) frontend-prod-cert

frontend-down:
	-@$(COMPOSE) stop $(FRONTEND_SERVICES)
	-@$(COMPOSE) rm -f $(FRONTEND_SERVICES)

frontend-dev-down:
	-@$(COMPOSE) stop frontend-dev
	-@$(COMPOSE) rm -f frontend-dev

frontend-prod-down:
	-@$(COMPOSE) stop frontend-prod
	-@$(COMPOSE) rm -f frontend-prod

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

k6-matchmaking:
	@$(K6) run $(K6_ARGS) \
		-e BASE_URL=$(K6_BASE_URL) \
		-e LOGIN=$(K6_LOGIN) \
		-e PASSWORD=$(K6_PASSWORD) \
		-e ACCESS_TOKEN=$(K6_ACCESS_TOKEN) \
		-e STRICT=$(K6_STRICT) \
		-e SLEEP_MS=$(K6_SLEEP_MS) \
		-e GAME_ID=$(K6_GAME_ID) \
		-e MIN_REGISTRATION_PRICE=$(K6_MIN_REGISTRATION_PRICE) \
		-e MAX_REGISTRATION_PRICE=$(K6_MAX_REGISTRATION_PRICE) \
		-e MIN_CAPACITY=$(K6_MIN_CAPACITY) \
		-e MAX_CAPACITY=$(K6_MAX_CAPACITY) \
		-e IS_BOOST=$(K6_IS_BOOST) \
		-e MIN_BOOST_POWER=$(K6_MIN_BOOST_POWER) \
		-e PAGE=$(K6_PAGE) \
		-e PAGE_SIZE=$(K6_PAGE_SIZE) \
		-e ENABLE_QUICK_MATCH=$(K6_ENABLE_QUICK_MATCH) \
		$(K6_DIR)/matchmaking.js

k6-game-server:
	@$(K6) run $(K6_ARGS) \
		-e BASE_URL=$(K6_BASE_URL) \
		-e LOGIN=$(K6_LOGIN) \
		-e PASSWORD=$(K6_PASSWORD) \
		-e ACCESS_TOKEN=$(K6_ACCESS_TOKEN) \
		-e STRICT=$(K6_STRICT) \
		-e SLEEP_MS=$(K6_SLEEP_MS) \
		-e GAME_ID=$(K6_GAME_ID) \
		-e MIN_REGISTRATION_PRICE=$(K6_MIN_REGISTRATION_PRICE) \
		-e MAX_REGISTRATION_PRICE=$(K6_MAX_REGISTRATION_PRICE) \
		-e MIN_CAPACITY=$(K6_MIN_CAPACITY) \
		-e MAX_CAPACITY=$(K6_MAX_CAPACITY) \
		-e IS_BOOST=$(K6_IS_BOOST) \
		-e MIN_BOOST_POWER=$(K6_MIN_BOOST_POWER) \
		-e ROOM_ID=$(K6_ROOM_ID) \
		-e ROUND_ID=$(K6_ROUND_ID) \
		-e LEAVE_AFTER=$(K6_LEAVE_AFTER) \
		$(K6_DIR)/game_server.js

k6-all: k6-matchmaking k6-game-server

down:
	@$(COMPOSE) down
