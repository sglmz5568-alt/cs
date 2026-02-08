#!/bin/bash

# SunnyProxy 启动脚本

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

if [ ! -f "./sunnyproxy" ]; then
    echo "编译 SunnyProxy..."
    go build -o sunnyproxy ./cmd/server
fi

CONFIG_FILE="${1:-configs/config.yaml}"

echo "启动 SunnyProxy..."
echo "配置文件: $CONFIG_FILE"

./sunnyproxy -config "$CONFIG_FILE"
