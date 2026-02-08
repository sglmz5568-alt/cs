#!/bin/bash

# SunnyProxy CA 证书安装脚本 (Linux)

PROXY_HOST="${1:-localhost}"
PROXY_PORT="${2:-2022}"

CERT_URL="http://${PROXY_HOST}:${PROXY_PORT}/ssl"
CERT_FILE="/tmp/SunnyProxy.cer"

echo "下载 CA 证书..."
curl -o "$CERT_FILE" "$CERT_URL"

if [ ! -f "$CERT_FILE" ]; then
    echo "下载失败，请检查代理服务是否运行"
    exit 1
fi

echo "证书已下载到: $CERT_FILE"

# 检测系统类型
if [ -f /etc/debian_version ]; then
    # Debian/Ubuntu
    echo "检测到 Debian/Ubuntu 系统"
    sudo cp "$CERT_FILE" /usr/local/share/ca-certificates/SunnyProxy.crt
    sudo update-ca-certificates
elif [ -f /etc/redhat-release ]; then
    # CentOS/RHEL
    echo "检测到 CentOS/RHEL 系统"
    sudo cp "$CERT_FILE" /etc/pki/ca-trust/source/anchors/SunnyProxy.crt
    sudo update-ca-trust
else
    echo "未知系统，请手动安装证书"
    echo "证书位置: $CERT_FILE"
fi

echo ""
echo "========================================"
echo "Linux 证书安装完成"
echo ""
echo "手机证书安装步骤:"
echo "1. 手机浏览器访问: $CERT_URL"
echo "2. 下载并安装证书"
echo "3. iOS: 设置 -> 通用 -> 关于本机 -> 证书信任设置 -> 信任证书"
echo "4. Android: 设置 -> 安全 -> 从存储设备安装证书"
echo "========================================"
