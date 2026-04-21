help:
	@echo "up-dev"
	@echo "up-prod"
	@echo "down"
	@echo "down-dev"
	@echo "down-prod"


up-prod:
	@docker compose --profile prod up -d

up-dev:
	@docker compose --profile dev up -d

down:
	@docker compose --profile dev --profile prod down

down-prod:
	@docker compose --profile prod down

down-dev:
	@docker compose --profile dev down
