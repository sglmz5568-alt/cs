# SunnyProxy 使用指南

## 项目简介

SunnyProxy 是一个远程抓包代理工具，支持：
- HTTP/HTTPS 请求拦截和修改
- URL 替换规则
- Header 修改
- Token 提取
- 实时日志查看
- Web 管理界面

## 目录结构

```
sunnyproxy/
├── cmd/server/main.go      # 主程序入口
├── configs/config.yaml     # 配置文件
├── internal/
│   ├── proxy/              # 代理核心
│   ├── rules/              # 规则引擎
│   ├── web/                # Web 管理界面
│   └── logger/             # 日志广播
├── ca.crt                  # CA 证书（手机需安装）
├── ca.key                  # CA 私钥
├── sunnyproxy.exe          # Windows 可执行文件
├── Dockerfile              # Docker 部署
└── bore.exe                # 隧道工具
```

## 快速启动

### 1. 本地启动服务

```bash
# 进入项目目录
cd C:/Users/Administrator/Desktop/2222/sunnyproxy

# 以管理员权限启动（重要！）
# 方法1：右键 sunnyproxy.exe -> 以管理员身份运行
# 方法2：PowerShell 管理员模式
powershell -Command "Start-Process -FilePath './sunnyproxy.exe' -Verb RunAs"
```

### 2. 创建公网隧道（使用 bore）

```bash
# 进入目录
cd C:/Users/Administrator/Desktop/2222

# 创建代理端口隧道
./bore.exe local 18888 --to bore.pub

# 创建 Web 管理端口隧道（新开一个终端）
./bore.exe local 18080 --to bore.pub
```

隧道会显示分配的公网端口，例如：
```
listening at bore.pub:5937   # 代理端口
listening at bore.pub:56705  # Web 管理端口
```

### 3. 手机设置

#### 代理设置
- 服务器：`bore.pub`
- 端口：隧道分配的代理端口（如 `5937`）

#### 证书安装（必须）

**iPhone:**
1. Safari 访问：`http://[Web管理地址]/ssl` 或 GitHub raw 地址
2. 设置 → 通用 → VPN与设备管理 → 安装证书
3. 设置 → 通用 → 关于本机 → 证书信任设置 → 开启信任

**Android:**
1. 浏览器访问证书下载地址
2. 设置 → 安全 → 加密与凭据 → 安装证书

## 配置文件说明

`configs/config.yaml`:

```yaml
server:
  proxy_port: 18888     # 代理服务端口
  web_port: 18080       # Web 管理端口
  bind_ip: "0.0.0.0"    # 绑定 IP

security:
  enabled: false        # 是否启用认证
  api_token: "xxx"      # API Token
  allowed_ips:
    - "0.0.0.0/0"

logging:
  level: "info"
  console: true
```

## 注意事项

### 必须以管理员权限运行
- Windows 需要管理员权限才能正确监听端口
- 普通权限启动可能导致服务失败

### 端口冲突问题
- 如果端口被占用，修改 `config.yaml` 中的端口号
- 使用 `netstat -an | grep 端口号` 检查端口占用

### 隧道不稳定
- bore.pub 是免费公共服务，可能会断开
- 断开后需要重新运行 bore 命令
- 每次重启端口号会变化

### 速度慢的解决方案
1. **局域网使用**：如果手机和电脑在同一 WiFi，直接用电脑 IP
   - 代理地址：`192.168.x.x:18888`
   - Web 管理：`http://192.168.x.x:18080`

2. **更换隧道服务**：bore.pub 服务器可能较远，可尝试其他服务

3. **部署到云服务器**：租用国内云服务器，延迟更低

### HTTPS 抓包必须安装证书
- 不安装证书，HTTPS 网站无法访问
- 证书文件：`ca.crt`
- 证书下载地址：`http://[Web管理地址]/ssl`

## 一键启动脚本

创建 `start.bat`：

```batch
@echo off
cd /d C:\Users\Administrator\Desktop\2222\sunnyproxy

:: 杀掉旧进程
taskkill /F /IM sunnyproxy.exe 2>nul
taskkill /F /IM bore.exe 2>nul
timeout /t 2

:: 启动代理服务（管理员权限）
powershell -Command "Start-Process -FilePath './sunnyproxy.exe' -Verb RunAs -WindowStyle Hidden"
timeout /t 3

:: 启动隧道
cd ..
start /b bore.exe local 18888 --to bore.pub
start /b bore.exe local 18080 --to bore.pub

echo 服务已启动，请查看隧道输出获取公网端口
pause
```

## 常用命令

```bash
# 检查服务状态
tasklist | grep sunnyproxy
netstat -an | grep 18888

# 查看日志
cat sunnyproxy.log

# 测试代理
curl -x http://bore.pub:端口 http://httpbin.org/ip
```

## 规则配置示例

在 Web 管理界面添加规则：

**URL 替换规则：**
- 匹配：`miniapp.qmai.cn`
- 替换为：`webapi2.qmai.cn`

**路径替换：**
- 匹配：`payment-another-info`
- 替换为：`payment-info`

**Token 提取：**
- Header 名：`Qm-User-Token`

## GitHub 仓库

https://github.com/sglmz5568-alt/cs
