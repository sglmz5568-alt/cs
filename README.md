# SunnyProxy - 远程抓包代理服务器

基于 Go 语言构建的远程抓包代理工具，支持 HTTP/HTTPS 请求拦截和修改。采用 [goproxy](https://github.com/elazarl/goproxy) 作为核心代理引擎，参考 [SunnyNet](https://gitee.com/qtr/SunnyNet) 的设计理念。

## 功能特性

- ✅ HTTP/HTTPS 代理抓包
- ✅ URL 替换规则
- ✅ 请求头修改
- ✅ Token 自动提取
- ✅ Web 管理界面
- ✅ 实时日志推送 (WebSocket)
- ✅ 规则热更新
- ✅ API Token 认证

## 快速开始

### 方式一：直接运行

```bash
# 1. 进入项目目录
cd sunnyproxy

# 2. 编译
go build -o sunnyproxy ./cmd/server

# 3. 运行
./sunnyproxy -config configs/config.yaml
```

### 方式二：Docker 部署

```bash
# 1. 构建镜像
docker build -t sunnyproxy .

# 2. 运行容器
docker run -d \
  -p 2021:2021 \
  -p 2022:2022 \
  -v $(pwd)/configs:/app/configs \
  --name sunnyproxy \
  sunnyproxy

# 或使用 docker-compose
docker-compose up -d
```

## 配置说明

配置文件位于 `configs/config.yaml`：

```yaml
server:
  proxy_port: 2021      # 代理服务端口
  web_port: 2022        # Web 管理端口
  bind_ip: "0.0.0.0"    # 绑定 IP

security:
  enabled: true                    # 是否启用认证
  api_token: "your-secret-token"   # API 认证 Token
  allowed_ips:                     # IP 白名单
    - "0.0.0.0/0"

logging:
  level: "info"         # 日志级别
  console: true         # 控制台输出

rules:
  file: "rules.json"    # 规则持久化文件
```

## 手机配置步骤

### 1. 安装 CA 证书

**iOS 设备：**
1. Safari 访问 `http://服务器IP:2022/ssl`
2. 下载并安装描述文件
3. 设置 → 通用 → 关于本机 → 证书信任设置
4. 启用对证书的完全信任

**Android 设备：**
1. 浏览器访问 `http://服务器IP:2022/ssl`
2. 下载证书文件
3. 设置 → 安全 → 从存储设备安装证书
4. 选择下载的证书文件安装

### 2. 配置 WiFi 代理

1. 进入 WiFi 设置
2. 点击已连接的网络 → 配置代理
3. 选择"手动"
4. 服务器：`服务器IP`
5. 端口：`2021`
6. 保存

### 3. 开始抓包

配置完成后，手机的 HTTP/HTTPS 请求将通过代理服务器，可在 Web 管理界面实时查看。

## Web 管理界面

访问 `http://服务器IP:2022` 进入管理界面。

如果启用了认证，需要在 URL 添加 token 参数：
```
http://服务器IP:2022?token=your-secret-token
```

### 功能说明

- **实时日志**：查看所有经过代理的请求
- **规则管理**：启用/禁用替换规则
- **Token 提取**：自动提取指定请求头的值
- **证书下载**：一键下载 CA 证书

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/rules | 获取所有规则 |
| POST | /api/rules | 创建规则 |
| PUT | /api/rules/:id | 更新规则 |
| DELETE | /api/rules/:id | 删除规则 |
| GET | /api/tokens | 获取提取的 Token |
| GET | /api/status | 服务状态 |
| GET | /ssl | 下载 CA 证书 |
| WebSocket | /api/logs/ws | 实时日志 |

### 认证方式

在请求头添加：
```
X-API-Token: your-secret-token
```

或在 URL 添加查询参数：
```
?token=your-secret-token
```

## 预置规则

项目预置了以下规则（基于原 E 语言逻辑）：

| 规则名称 | 类型 | 匹配 | 替换 |
|---------|------|------|------|
| qmai 域名替换 | url_replace | miniapp.qmai.cn | webapi2.qmai.cn |
| payment 路径替换 | url_replace | payment-another-info | payment-info |
| 提取 Qm-User-Token | token_extract | Qm-User-Token | - |

## 目录结构

```
sunnyproxy/
├── cmd/server/main.go          # 入口文件
├── internal/
│   ├── proxy/                  # 代理核心
│   ├── rules/                  # 规则引擎
│   ├── web/                    # Web 管理
│   └── logger/                 # 日志广播
├── pkg/config/                 # 配置管理
├── configs/                    # 配置文件
├── scripts/                    # 脚本
├── Dockerfile
├── docker-compose.yaml
└── README.md
```

## 注意事项

1. **跨平台支持**：基于 goproxy，支持 Windows、Linux、macOS
2. **证书信任**：手机必须安装并信任 CA 证书才能抓取 HTTPS 流量
3. **安全建议**：生产环境请修改默认 API Token，建议配置 IP 白名单
4. **防火墙**：确保服务器开放 2021 和 2022 端口

## 故障排除

### 手机无法连接代理
- 检查服务器防火墙是否开放端口
- 确认手机和服务器在同一网络或可互通
- 检查代理配置是否正确

### HTTPS 请求失败
- 确认已安装 CA 证书
- iOS 需要在"证书信任设置"中启用完全信任
- 部分 App 可能有证书固定 (SSL Pinning)，无法抓包

### Web 界面无法访问
- 检查 2022 端口是否开放
- 如果启用了认证，确认 token 正确

## 许可证

本项目基于 SunnyNet，遵循 MIT 许可证。
