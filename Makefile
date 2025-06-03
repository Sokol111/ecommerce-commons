.PHONY: generate generate-mocks start-docker-mongo start-local-mongo stop-mongo stop-local-mongo start-kafka stop-kafka update-dependencies test init-git

generate-mocks:
	mockery

start-docker-mongo: stop-mongo
	docker network inspect shared-network || docker network create shared-network
	MONGO_HOST=mongo docker compose -f ./infrastructure/docker/mongo.yml up -d

start-local-mongo: stop-mongo
	docker network inspect shared-network || docker network create shared-network
	MONGO_HOST=localhost docker compose -f ./infrastructure/docker/mongo.yml up -d

stop-mongo:
	docker compose -f ./infrastructure/docker/mongo.yml down

start-kafka:
	docker compose -f ./infrastructure/docker/kafka.yml up -d

stop-kafka:
	docker compose -f ./infrastructure/docker/kafka.yml down -v

update-dependencies:
	go get -u ./...

test:
	go test ./... -v -cover

init-git:
	git config user.name "Sokol111"
	git config user.email "igorsokol111@gmail.com"
	git config commit.gpgSign false