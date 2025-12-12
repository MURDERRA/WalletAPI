FROM golang:1.25.5-alpine AS builder
# Образ сборки
WORKDIR /build

RUN apk update --no-cache && apk add --no-cache tzdata

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest

RUN swag init -g ./main.go --parseInternal

RUN CGO_ENABLED=0 GOOS=linux go build -o /build/wallets-api ./main.go

# Финальный образ, в котором только исполняемый файл - так финальный образ легче по весу
FROM alpine:latest
WORKDIR /app

RUN apk update --no-cache && apk add --no-cache ca-certificates

COPY --from=builder /build/wallets-api /app/wallets-api

COPY config.env /app/config.env

# Делаем файл исполняемым
RUN chmod +x /app/wallets-api

EXPOSE 8080

CMD ["/app/wallets-api"]