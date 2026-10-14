.PHONY: test backend frontend embed build

test:
	go test ./...

backend:
	go run ./cmd/server

frontend:
	cd frontend && npm run dev

embed:
	bash scripts/embed-vue.sh

build: embed
	go build -o bin/etlgo ./cmd/server
