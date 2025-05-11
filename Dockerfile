FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY FileManager/ .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/gateway ./cmd/main/gateway/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/app ./cmd/main/app/main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bin .

CMD ["/app/app", "/app/gateway"]
