FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o sunnyproxy ./cmd/server

# 下载 frp 客户端
RUN wget -O frp.tar.gz https://github.com/fatedier/frp/releases/download/v0.61.1/frp_0.61.1_linux_amd64.tar.gz \
    && tar -xzf frp.tar.gz \
    && mv frp_0.61.1_linux_amd64/frpc /app/frpc \
    && chmod +x /app/frpc

# 下载 bore 隧道工具
RUN wget -O bore.tar.gz https://github.com/ekzhang/bore/releases/download/v0.5.2/bore-v0.5.2-x86_64-unknown-linux-musl.tar.gz \
    && tar -xzf bore.tar.gz \
    && chmod +x bore

FROM alpine:3.18

RUN apk add --no-cache ca-certificates bash

WORKDIR /app

COPY --from=builder /app/sunnyproxy /app/sunnyproxy
COPY --from=builder /app/frpc /app/frpc
COPY --from=builder /app/bore /app/bore
COPY --from=builder /app/configs /app/configs

# frpc 配置
COPY frpc.toml /app/frpc.toml

# 启动脚本
COPY start.sh /app/start.sh
RUN chmod +x /app/start.sh

EXPOSE 8888 8080

CMD ["/app/start.sh"]
