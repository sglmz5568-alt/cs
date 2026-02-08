#!/bin/bash

# 启动 SunnyProxy
/app/sunnyproxy -config /app/configs/config.yaml &

# 等待服务启动
sleep 3

# 自动重连函数
start_tunnel() {
    while true; do
        echo "Starting bore tunnel for port $1..."
        /app/bore local $1 --to bore.pub
        echo "Tunnel for port $1 disconnected, reconnecting in 5 seconds..."
        sleep 5
    done
}

# 启动隧道（带自动重连）
start_tunnel 8888 &
start_tunnel 8080 &

# 保持容器运行
wait
