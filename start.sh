#!/bin/bash

# 启动 SunnyProxy
/app/sunnyproxy -config /app/configs/config.yaml &
PROXY_PID=$!

# 等待服务启动
sleep 3

# 定时清理函数
cleanup_task() {
    while true; do
        sleep 3600
        echo "[Cleanup] Running cleanup at $(date)"
        rm -rf /tmp/* 2>/dev/null
        find /app -name "*.log" -mtime +1 -delete 2>/dev/null
        echo "[Cleanup] Cleanup completed"
    done
}

# 健康检查函数
health_check() {
    while true; do
        sleep 300
        if ! kill -0 $PROXY_PID 2>/dev/null; then
            echo "[HealthCheck] SunnyProxy crashed, restarting..."
            /app/sunnyproxy -config /app/configs/config.yaml &
            PROXY_PID=$!
        fi
    done
}

# 启动清理和健康检查
cleanup_task &
health_check &

# 检查是否有 ngrok token
if [ -n "$NGROK_AUTHTOKEN" ]; then
    echo "Using ngrok with fixed domain..."
    /app/ngrok config add-authtoken $NGROK_AUTHTOKEN

    # 如果有自定义域名
    if [ -n "$NGROK_DOMAIN" ]; then
        /app/ngrok tcp 8080 --domain=$NGROK_DOMAIN &
    else
        /app/ngrok tcp 8080 --log=stdout &
    fi
else
    # 使用 bore（端口不固定）
    echo "Using bore tunnel (port will change on restart)..."
    while true; do
        echo "Starting bore tunnel..."
        /app/bore local 8080 --to bore.pub 2>&1
        echo "Bore disconnected, reconnecting in 5 seconds..."
        sleep 5
    done
fi

# 保持运行
wait
