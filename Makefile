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

SWAGGER_UI_VERSION=5.19.0
OPENAPI_YML_URL=https://raw.githubusercontent.com/canpok1/plant-diary/refs/heads/main/docs/openapi.yaml

.PHONY: doc-api
doc-api:
	mkdir -p docs/api
	curl -L https://github.com/swagger-api/swagger-ui/archive/refs/tags/v$(SWAGGER_UI_VERSION).zip -o docs/api/swagger-ui.zip
	unzip docs/api/swagger-ui.zip -d docs/api
	cp -R docs/api/swagger-ui-$(SWAGGER_UI_VERSION)/dist/* docs/api/
	sed -i 's@url:.*@url: "$(OPENAPI_YML_URL)",@g' docs/api/swagger-initializer.js
	rm -r docs/api/swagger-ui-$(SWAGGER_UI_VERSION)
	rm docs/api/swagger-ui.zip

.PHONY: doc-db
doc-db:
	mkdir -p tmp
	for f in $$(ls migrations/*.up.sql | sort); do sqlite3 tmp/schema.db < $$f; done
	tbls doc --force
	rm -f tmp/schema.db
	@rmdir --ignore-fail-on-non-empty tmp 2>/dev/null || true
