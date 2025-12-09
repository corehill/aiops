app:
	cd .. && sudo docker build -t app:v1 . && cd sre

up:
	sudo docker compose up -d

down:
	sudo docker compose down

logs:
	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		echo "📜 查看所有docker compose服务日志..."; \
		sudo docker compose logs; \
	else \
		SERVICE=$(filter-out $@,$(MAKECMDGOALS)); \
		echo "📜 查看 $$SERVICE 服务日志..."; \
		sudo docker compose logs $$SERVICE; \
	fi

.PHONY: app up down logs