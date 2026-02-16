.PHONY: setup
setup:
	go mod download

.PHONY: fmt-check
fmt-check:
	test -z "$$(gofmt -l .)"

.PHONY: lint
lint:
	go vet ./...

.PHONY: test
test:
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: build
build:
	go build -v ./...
