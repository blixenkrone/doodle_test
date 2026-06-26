.PHONY: build build_image install install_dev_tools generate_docs generate_sqlc run test test_unit lint format format_go format_sql scan new_migration git_hook

app_name?=corti
arch?=amd64 # amd64, arm64
os?=linux
image_name?=$(app_name):local-${os}-${arch}
golangci_lint_version?=2.1.6
migration_name?=

build:
	GOOS=$(os) GOARCH=$(arch) CGO_ENABLED=0 go build -ldflags="-w -s" -o ./bin/$(app_name)-$(os)-$(arch) cmd/main.go

build_image:
	docker buildx build --platform $(os)/$(arch) -t $(image_name) .

install:
	go mod tidy

install_dev_tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(golangci_lint_version)
	go install golang.org/x/vuln/cmd/govulncheck@latest

generate_docs:
	swag init --quiet --parseDependency --parseInternal -g cmd/main.go --output docs/

generate_sqlc:
	@echo "running SQLC generate" && \
	docker run --rm -v $$(pwd)/:/src -w /src sqlc/sqlc generate && \
	echo "finished SQLC generate"

run:
	go run cmd/main.go

# test runs the full suite including docker-backed integration tests.
test:
	go clean -testcache && go test -covermode atomic -cover -race ./...

# test_unit skips the docker-backed integration package.
test_unit:
	go test -race $$(go list ./... | grep -v /integration)

lint:
	golangci-lint run --allow-parallel-runners --timeout=60s --disable errcheck && go vet ./...

format_go:
	goimports -l -w ./
	gofmt -l -w -s ./

format_sql:
	docker run --rm -v $$(pwd):/sql sqlfluff/sqlfluff:3.4.0 fix --quiet --config .sqlfluff --dialect postgres .

format: format_go format_sql

scan:
	govulncheck ./...

new_migration:
	@echo "creating migration $(migration_name)"
	migrate create -ext sql -dir ./internal/storage/sql/migrations -seq $(migration_name)

git_hook: generate_docs generate_sqlc format lint install scan
	@echo "Running git pre commit hook"
