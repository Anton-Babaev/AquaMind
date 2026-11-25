.PHONY: test test-race cover lint build build-pi run-edge run-backend tidy clean

# Один модуль, две программы в cmd/: edge и backend.
# Frontend — часть Backend-контура: статика встроена в бинарник backend
# (backend/delivery/http/static через //go:embed), отдельного cmd/frontend нет.
BIN_DIR := bin

test:
	go test ./...

# Тесты под race-детектором: ловит гонки в горутинах (poller, syncworker, control).
test-race:
	go test -race ./...

# Покрытие: coverage.out + coverage.html для отчёта.
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	go vet ./...

build:
	go build -o $(BIN_DIR)/edge ./edge/cmd/edge
	go build -o $(BIN_DIR)/backend ./backend/cmd/backend
# Кросс-компиляция под Raspberry Pi 3 B+ (ARMv7).
build-pi:
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BIN_DIR)/edge-pi ./edge/cmd/edge

run-edge:
	go run ./edge/cmd/edge

run-backend:
	go run ./backend/cmd/backend

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)

