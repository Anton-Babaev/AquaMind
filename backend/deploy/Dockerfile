# Dockerfile для Backend-сервиса AquaMind.
#
# Многоступенчатая сборка: бинарник собирается в golang-образе, а в финальный
# минимальный образ кладётся только он + конфиг.
#
# Сборка:  docker build -t aquamind-backend -f backend/deploy/Dockerfile .
# Запуск:  docker run -p 8080:8080 --env-file .env aquamind-backend

# --- Стадия сборки ---
FROM golang:1.26-alpine AS build
WORKDIR /src

# Кэшируем зависимости отдельно от исходников.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/backend

# --- Финальная стадия ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=build /out/backend /app/backend
COPY configs/backend.yaml /app/configs/backend.yaml

# Секреты (DSN, токен) передаются через переменные окружения, не в образе.
ENV AQUAMIND_BACKEND_LISTEN=":8080"
EXPOSE 8080

ENTRYPOINT ["/app/backend"]
CMD ["--config", "/app/configs/backend.yaml"]
