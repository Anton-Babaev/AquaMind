.PHONY: test lint build build-pi run-edge run-backend run-frontend tidy clean

# Один модуль, три программы в cmd/.
BIN_DIR := bin

test:
	go test ./...

lint:
	go vet ./...

build:
	go build -o $(BIN_DIR)/edge ./cmd/edge
	go build -o $(BIN_DIR)/backend ./cmd/backend
	go build -o $(BIN_DIR)/frontend ./cmd/frontend

# Кросс-компиляция под Raspberry Pi 3 B+ (ARMv7).
build-pi:
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BIN_DIR)/edge-pi ./cmd/edge

run-edge:
	go run ./cmd/edge

run-backend:
	go run ./cmd/backend

run-frontend:
	go run ./cmd/frontend

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
