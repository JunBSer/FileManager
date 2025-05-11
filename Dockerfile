FROM golang:1.24-alpine AS builder
LABEL maintainer="JunBser"

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/gateway ./FileManager/cmd/main/gateway/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/app     ./FileManager/cmd/main/app/main.go

FROM alpine:3.19

WORKDIR /app

COPY FileManager/configs/local.env configs/local.env

COPY --from=builder /build/bin/gateway ./gateway
COPY --from=builder /build/bin/app     ./app

COPY entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
