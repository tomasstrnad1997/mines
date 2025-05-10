.PHONY: db test
include .env
export
test:
	go test ./db ./protocol ./matchmaking ./protocol ./gamelauncher

db:
	go run ./cmd/db
