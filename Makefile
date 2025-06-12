.PHONY: generate-mocks update-dependencies test

generate-mocks:
	mockery

update-dependencies:
	go get -u ./...

test:
	go test ./... -v -cover
