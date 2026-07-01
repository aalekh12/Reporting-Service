.PHONY: run migrate-up migrate-down test test-race cover build

# Pin to the locally installed Go toolchain. Without this, `go` auto-downloads
# whatever toolchain version a dependency's go.mod asks for, and that
# downloaded toolchain has been observed missing bundled tools (e.g. covdata)
# needed by `go test -coverprofile` on Windows.
export GOTOOLCHAIN=local

run:
	go run ./cmd/server

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

# Requires cgo (a C compiler on PATH); omitted from the default `test`
# target since it isn't available in every dev environment.
test-race:
	go test ./... -race

cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out
