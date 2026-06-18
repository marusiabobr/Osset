APP=lingw

.PHONY: run test build tidy migrate-up migrate-down

run:
	go run ./cmd/lingw

test:
	go test ./...

build:
	go build -o bin/$(APP) ./cmd/lingw

tidy:
	go mod tidy

migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down 1
