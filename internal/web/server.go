package web

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"sunnyproxy/internal/proxy"
	"sunnyproxy/internal/rules"
	"sunnyproxy/pkg/config"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	config  *config.Config
	api     *API
	ws      *WSHandler
	auth    *AuthMiddleware
	engine  *rules.Engine
	wrapper *proxy.Wrapper
}

func NewServer(cfg *config.Config, engine *rules.Engine, wrapper *proxy.Wrapper) *Server {
	return &Server{
		config:  cfg,
		api:     NewAPI(engine, wrapper),
		ws:      NewWSHandler(),
		auth:    NewAuthMiddleware(cfg),
		engine:  engine,
		wrapper: wrapper,
	}
}

func (s *Server) GetHandler() http.Handler {
	mux := http.NewServeMux()

	s.api.RegisterRoutes(mux)

	mux.HandleFunc("/api/logs/ws", s.ws.HandleWebSocket)

	staticFS, err := fs.Sub(staticFiles, "static")
	if err == nil {
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	mux.HandleFunc("/", s.handleIndex)

	return s.auth.Handler(mux)
}

func (s *Server) Start() error {
	handler := s.GetHandler()

	addr := fmt.Sprintf("%s:%d", s.config.Server.BindIP, s.config.Server.WebPort)
	log.Printf("Web management interface starting on http://%s\n", addr)
	return http.ListenAndServe(addr, handler)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SunnyProxy - 代理管理</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #eee; min-height: 100vh; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        header { background: #16213e; padding: 20px; border-radius: 10px; margin-bottom: 20px; }
        h1 { font-size: 24px; color: #00d9ff; }
        .status { display: inline-block; background: #00ff88; color: #000; padding: 4px 12px; border-radius: 20px; font-size: 12px; margin-left: 10px; }
        .grid { display: grid; grid-template-columns: 2fr 1fr; gap: 20px; }
        @media (max-width: 768px) { .grid { grid-template-columns: 1fr; } }
        .card { background: #16213e; border-radius: 10px; padding: 20px; }
        .card h2 { font-size: 18px; margin-bottom: 15px; color: #00d9ff; }
        .log-container { height: 500px; overflow-y: auto; background: #0f0f23; border-radius: 5px; padding: 10px; font-family: monospace; font-size: 13px; }
        .log-entry { padding: 5px 0; border-bottom: 1px solid #333; }
        .log-entry.request { color: #00ff88; }
        .log-entry.response { color: #00d9ff; }
        .log-entry.error { color: #ff6b6b; }
        .log-entry.token { color: #ffd93d; }
        .log-time { color: #888; margin-right: 10px; }
        .log-modified { background: #ff6b6b; color: #fff; padding: 2px 6px; border-radius: 3px; font-size: 10px; margin-left: 5px; }
        .btn { background: #00d9ff; color: #000; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #00b8d9; }
        .token-list { max-height: 300px; overflow-y: auto; }
        .token-item { background: #0f0f23; padding: 10px; border-radius: 5px; margin-bottom: 10px; }
        .token-value { font-family: monospace; word-break: break-all; color: #ffd93d; font-size: 12px; }
        .copy-btn { background: #333; border: none; color: #fff; padding: 4px 8px; border-radius: 3px; cursor: pointer; font-size: 12px; margin-top: 5px; }
        .copy-btn:hover { background: #444; }
        .info { background: #0f0f23; padding: 15px; border-radius: 5px; margin-top: 15px; }
        .info p { margin: 5px 0; color: #888; }
        .info code { background: #333; padding: 2px 6px; border-radius: 3px; color: #00d9ff; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>SunnyProxy <span class="status" id="status">连接中...</span></h1>
        </header>

        <div class="grid">
            <div class="card">
                <h2>实时日志</h2>
                <div class="log-container" id="logs"></div>
            </div>

            <div class="card">
                <h2>提取的 Token</h2>
                <div class="token-list" id="tokens"></div>

                <div class="info">
                    <p><strong>代理配置</strong></p>
                    <p>服务器: <code id="proxy-host">-</code></p>
                    <p>端口: <code id="proxy-port">-</code></p>
                    <p><a href="/ssl" class="btn" style="margin-top: 10px; display: inline-block; text-decoration: none;">下载 CA 证书</a></p>
                </div>
            </div>
        </div>
    </div>

    <script>
        let ws;
        function connectWS() {
            const wsUrl = (location.protocol === 'https:' ? 'wss:' : 'ws:') + '//' + location.host + '/api/logs/ws';
            ws = new WebSocket(wsUrl);
            ws.onopen = () => {
                document.getElementById('status').textContent = '运行中';
                document.getElementById('status').style.background = '#00ff88';
            };
            ws.onclose = () => {
                document.getElementById('status').textContent = '已断开';
                document.getElementById('status').style.background = '#ff6b6b';
                setTimeout(connectWS, 3000);
            };
            ws.onmessage = (e) => {
                const log = JSON.parse(e.data);
                addLog(log);
            };
        }

        function addLog(log) {
            const container = document.getElementById('logs');
            const div = document.createElement('div');
            div.className = 'log-entry ' + log.type;
            const time = new Date(log.timestamp).toLocaleTimeString();
            let content = '<span class="log-time">' + time + '</span>';
            if (log.type === 'request') {
                content += '[' + log.method + '] ' + log.url;
                if (log.modified) content += '<span class="log-modified">MODIFIED</span>';
            } else if (log.type === 'response') {
                content += '[' + log.status_code + '] ' + log.url;
            } else if (log.type === 'token') {
                content += '[TOKEN] ' + log.url;
            } else if (log.type === 'error') {
                content += '[ERROR] ' + log.error;
            }
            div.innerHTML = content;
            container.insertBefore(div, container.firstChild);
            if (container.children.length > 200) container.removeChild(container.lastChild);
        }

        async function loadTokens() {
            try {
                const res = await fetch('/api/tokens');
                const tokens = await res.json();
                const container = document.getElementById('tokens');
                container.innerHTML = '';
                tokens.slice(-5).reverse().forEach(t => {
                    const div = document.createElement('div');
                    div.className = 'token-item';
                    div.innerHTML = '<strong>' + t.name + '</strong><div class="token-value">' + t.value + '</div><button class="copy-btn" onclick="navigator.clipboard.writeText(\'' + t.value + '\')">复制</button>';
                    container.appendChild(div);
                });
            } catch (e) { console.error(e); }
        }

        async function loadStatus() {
            try {
                const res = await fetch('/api/status');
                const status = await res.json();
                document.getElementById('proxy-host').textContent = location.hostname;
                document.getElementById('proxy-port').textContent = status.proxy_port;
            } catch (e) { console.error(e); }
        }

        connectWS();
        loadTokens();
        loadStatus();
        setInterval(loadTokens, 5000);
    </script>
</body>
</html>`
