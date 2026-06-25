.PHONY: run migrate-up migrate-down migrate-status build

run:
	go run main.go

build:
	go build -o beacon-be main.go

migrate-up:
	go run cmd/migrate/main.go -command up

migrate-down:
	go run cmd/migrate/main.go -command down

migrate-status:
	go run cmd/migrate/main.go -command version

test:
	go test ./...

tidy:
	go mod tidy
