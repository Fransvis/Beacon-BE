.PHONY: run migrate-up migrate-down migrate-status migrate-version migrate-force build

run:
	go run main.go

build:
	go build -o beacon-be main.go

migrate-up:
	go run cmd/migrate/main.go -command up

migrate-down:
	go run cmd/migrate/main.go -command down

migrate-version:
	go run cmd/migrate/main.go -command version

migrate-force:
	go run cmd/migrate/main.go -command force -version $(v)

test:
	go test ./...

tidy:
	go mod tidy
