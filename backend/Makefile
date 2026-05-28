.PHONY: run build test migrate-up migrate-down

run:
	go run cmd/api/main.go

build:
	go build -o bin/api cmd/api/main.go
	go build -o bin/worker cmd/worker/main.go

test:
	go test ./... -v -timeout 120s

test-integration:
	go test ./tests/integration/... -v -timeout 120s

test-e2e:
	go test ./tests/e2e/... -v -timeout 180s

migrate-up:
	migrate -path migrations \
	-database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations \
	-database "$(DATABASE_URL)" down

migrate-create:
	migrate create -ext sql \
	-dir migrations \
	-seq $(name)

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker-build:
	docker build -t zipdesk-api .

docker-run:
	docker run -p 8080:8080 \
	--env-file .env \
	zipdesk-api
