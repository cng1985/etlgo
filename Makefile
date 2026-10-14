.PHONY: test backend frontend embed build

ifeq ($(OS),Windows_NT)
EMBED := cmd /c scripts\embed-vue.bat
else
EMBED := bash scripts/embed-vue.sh
endif

test:
	go test ./...

backend:
	go run ./cmd/server

frontend:
	cd frontend && npm run dev

embed:
	$(EMBED)

build: embed
	go build -o bin/etlgo ./cmd/server
