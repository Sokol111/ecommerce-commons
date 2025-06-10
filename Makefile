.PHONY: generate-mocks start-docker-mongo start-local-mongo stop-mongo start-kafka stop-kafka update-dependencies test init-git stop-and-delete-mongo ensure-network start-traefik stop-traefik

generate-mocks:
	mockery

ensure-network:
	docker network inspect shared-network > /dev/null 2>&1 || docker network create shared-network

start-docker-mongo: ensure-network stop-mongo
	MONGO_HOST=mongo docker compose -f ./infrastructure/docker/mongo.yml up -d

start-local-mongo: ensure-network stop-mongo
	MONGO_HOST=localhost docker compose -f ./infrastructure/docker/mongo.yml up -d

stop-mongo:
	docker compose -f ./infrastructure/docker/mongo.yml down

stop-and-delete-mongo:
	docker compose -f ./infrastructure/docker/mongo.yml down -v

start-kafka: ensure-network
	docker compose -f ./infrastructure/docker/kafka.yml up -d

stop-kafka:
	docker compose -f ./infrastructure/docker/kafka.yml down -v

start-traefik: ensure-network
	docker compose -f ./infrastructure/docker/traefik.yml up -d

stop-traefik:
	docker compose -f ./infrastructure/docker/traefik.yml down

update-dependencies:
	go get -u ./...

test:
	go test ./... -v -cover

init-git:
	git config user.name "Sokol111"
	git config user.email "igorsokol111@gmail.com"
	git config commit.gpgSign false