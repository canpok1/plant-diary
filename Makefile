.PHONY: setup
setup:
	go mod download

.PHONY: fmt-check
fmt-check:
	test -z "$$(gofmt -l .)"

.PHONY: lint
lint:
	go vet ./...

.PHONY: depcheck
depcheck:
	go vet -vettool=$(shell which depcheck) ./...

.PHONY: test
test:
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-e2e
test-e2e:
	go test -v -tags e2e -race ./...

.PHONY: build
build:
	go build -v ./...

.PHONY: doc-db
doc-db:
	mkdir -p tmp
	for f in $$(ls migrations/*.up.sql | sort); do sqlite3 tmp/schema.db < $$f; done
	tbls doc --force
	rm -f tmp/schema.db
	@rmdir --ignore-fail-on-non-empty tmp 2>/dev/null || true
