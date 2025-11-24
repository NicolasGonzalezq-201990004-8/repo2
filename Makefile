.PHONY: all clean docker-up docker-down docker-logs

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
