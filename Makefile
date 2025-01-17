create-mocks:
	mockery

start-mongo:
	docker compose -f ./infrastructure/docker/mongo.yml up -d

stop-mongo:
	docker compose -f /infrastructure/docker/mongo.yml down

start-kafka:
	docker compose -f ./infrastructure/docker/kafka.yml up -d

stop-kafka:
	docker compose -f /infrastructure/docker/kafka.yml down

test:
	go test ./... -v -cover
