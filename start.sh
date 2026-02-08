#!/bin/bash

# 启动 SunnyProxy（直接使用 Railway TCP 端口，无需隧道）
echo "Starting SunnyProxy..."
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

echo "SunnyProxy is running. Use Railway TCP proxy to connect."

# 保持容器运行
wait $PROXY_PID
