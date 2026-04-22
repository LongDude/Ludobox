help:
	@echo 'down - остановить все контейнеры'
	@echo "game1 - Клиент Redis для микросервера game-server-1"
	@echo "game2 - Клиент Redis для микросервера game-server-2"
	@echo "matchmaking - Клиент Redis для сервера организации матчей"
	@echo "user - Клиент Redis для сервера обработки пользователей"

down:
	@docker compose down

up:
	@docker compose up -d

game1:
	@docker compose up -d rc-game1
	@echo "Redis commander for game-redis-1: http://localhost:9001"

game2:
	@docker compose up -d rc-game2
	@echo "Redis commander for game-redis-2: http://localhost:9002"

matchmaking:
	@docker compose up -d rc-matchmaking 
	@echo "Redis commander for matchmaking-redis: http://localhost:9003"

user:
	@docker compose up -d rc-user 
	@echo "Redis commander for user-redis: http://localhost:9004"
