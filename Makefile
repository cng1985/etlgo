.PHONY: test backend frontend build

test:
	go test ./...

backend:
	go run ./cmd/server

frontend:
	cd frontend && npm run dev

build:
	cd frontend && npm run build
	go build -o bin/etlgo ./cmd/server
