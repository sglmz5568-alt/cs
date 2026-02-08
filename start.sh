#!/bin/bash

# 启动 SunnyProxy
/app/sunnyproxy -config /app/configs/config.yaml &

# 等待服务启动
sleep 3

# 启动 bore 隧道暴露代理端口
/app/bore local 8888 --to bore.pub &

# 启动 bore 隧道暴露 Web 端口
/app/bore local 8080 --to bore.pub &

# 保持容器运行
wait
