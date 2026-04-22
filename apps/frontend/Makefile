help:
	@echo "up-dev - запустить клиентскую часть в dev-окружении"
	@echo "up-prod - запустить клиентскую часть в рабочем окружении"
	@echo "down - завершить работу всех контейнеров клиентской части"
	@echo "down-dev - завершить работу контейнеров клиентской части dev-окружения"
	@echo "down-prod - завершить работу контейнеров клиентской части рабочего окружения"

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
