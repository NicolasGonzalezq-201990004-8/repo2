.PHONY: all clean docker-up docker-down docker-vm1 docker-vm2 docker-vm3 docker-vm4
.PHONY: docker-logs docker-logs-vm1 docker-logs-vm2 docker-logs-vm3 docker-logs-vm4

DOCKER_COMPOSE := docker-compose

clean: 

# ----------
# DOCKER
# ----------

docker-up:
	@$(DOCKER_COMPOSE) up -d --build

docker-down:
	@$(DOCKER_COMPOSE) down -v --remove-orphans

docker-logs:
	@$(DOCKER_COMPOSE) logs -f

docker-vm1:
	@$(DOCKER_COMPOSE) --profile vm1 up -d --build

docker-logs-vm1:
	@$(DOCKER_COMPOSE) --profile vm1 logs -f

docker-vm2:
	@$(DOCKER_COMPOSE) --profile vm2 up -d --build

docker-logs-vm2:
	@$(DOCKER_COMPOSE) --profile vm2 logs -f

docker-vm3:
	@$(DOCKER_COMPOSE) --profile vm3 up -d --build

docker-logs-vm3:
	@$(DOCKER_COMPOSE) --profile vm3 logs -f

docker-vm4:
	@$(DOCKER_COMPOSE) --profile vm4 up -d --build

docker-logs-vm4:
	@$(DOCKER_COMPOSE) --profile vm4 logs -f
