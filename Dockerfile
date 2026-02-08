FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o sunnyproxy ./cmd/server

FROM alpine:3.18

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/sunnyproxy /app/sunnyproxy
COPY --from=builder /app/configs /app/configs

EXPOSE 8888 8080

CMD ["/app/sunnyproxy", "-config", "/app/configs/config.yaml"]
