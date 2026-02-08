#!/bin/bash

# 启动 SunnyProxy
/app/sunnyproxy -config /app/configs/config.yaml &

# 等待服务启动
sleep 3

# 启动 frp 客户端（会自动重连）
echo "Starting frp tunnel..."
/app/frpc -c /app/frpc.toml &

# 备用：同时启动 bore 隧道
while true; do
    echo "Starting bore tunnel..."
    /app/bore local 8080 --to bore.pub 2>&1 || true
    echo "Bore disconnected, reconnecting in 5 seconds..."
    sleep 5
done
