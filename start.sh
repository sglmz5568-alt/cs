#!/bin/bash

# 启动 SunnyProxy
/app/sunnyproxy -config /app/configs/config.yaml &
PROXY_PID=$!

# 等待服务启动
sleep 3

# 启动 frp 客户端（会自动重连）
echo "Starting frp tunnel..."
/app/frpc -c /app/frpc.toml &

# 定时清理函数
cleanup_task() {
    while true; do
        sleep 3600  # 每小时执行一次
        echo "[Cleanup] Running cleanup at $(date)"

        # 清理临时文件
        rm -rf /tmp/* 2>/dev/null

        # 清理旧日志文件
        find /app -name "*.log" -mtime +1 -delete 2>/dev/null

        # 清理系统缓存
        sync && echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true

        echo "[Cleanup] Cleanup completed"
    done
}

# 健康检查函数
health_check() {
    while true; do
        sleep 300  # 每5分钟检查一次

        # 检查 SunnyProxy 是否还在运行
        if ! kill -0 $PROXY_PID 2>/dev/null; then
            echo "[HealthCheck] SunnyProxy crashed, restarting..."
            /app/sunnyproxy -config /app/configs/config.yaml &
            PROXY_PID=$!
        fi

        # 打印内存使用情况
        echo "[HealthCheck] Memory usage: $(free -m 2>/dev/null | grep Mem | awk '{print $3"MB/"$2"MB"}' || echo 'N/A')"
    done
}

# 启动清理任务
cleanup_task &

# 启动健康检查
health_check &

# 启动 bore 隧道（带自动重连）
while true; do
    echo "Starting bore tunnel..."
    /app/bore local 8080 --to bore.pub 2>&1 || true
    echo "Bore disconnected, reconnecting in 5 seconds..."
    sleep 5
done
