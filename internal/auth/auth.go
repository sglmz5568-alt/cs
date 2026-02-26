package auth

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/elazarl/goproxy"
)

type ProxyAuth struct {
	enabled  bool
	username string
	password string
	mu       sync.RWMutex
}

var (
	authInstance *ProxyAuth
	authOnce     sync.Once
)

func GetAuth() *ProxyAuth {
	authOnce.Do(func() {
		authInstance = &ProxyAuth{
			enabled: false,
		}
	})
	return authInstance
}

// SetCredentials 设置认证凭据
func (a *ProxyAuth) SetCredentials(username, password string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.username = username
	a.password = password
	a.enabled = true
	log.Printf("[Auth] 代理认证已启用，用户名: %s", username)
}

// SetEnabled 启用/禁用认证
func (a *ProxyAuth) SetEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = enabled
}

// IsEnabled 检查是否启用认证
func (a *ProxyAuth) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// Validate 验证用户名密码
func (a *ProxyAuth) Validate(username, password string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return username == a.username && password == a.password
}

// ParseBasicAuth 解析 Proxy-Authorization header
func ParseBasicAuth(authHeader string) (username, password string, ok bool) {
	if authHeader == "" {
		return "", "", false
	}

	// 格式: Basic base64(username:password)
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", "", false
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// RequireAuth 返回 407 要求认证的响应
func RequireAuth() *http.Response {
	resp := &http.Response{
		StatusCode: 407,
		Status:     "407 Proxy Authentication Required",
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	resp.Header.Set("Proxy-Authenticate", `Basic realm="SunnyProxy"`)
	resp.Header.Set("Content-Length", "0")
	return resp
}

// SetupAuth 在代理上设置认证中间件
func (a *ProxyAuth) SetupAuth(proxy *goproxy.ProxyHttpServer) {
	// 处理 CONNECT 请求的认证
	originalHandler := proxy.OnRequest()

	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if !a.IsEnabled() {
			return req, nil
		}

		authHeader := req.Header.Get("Proxy-Authorization")
		username, password, ok := ParseBasicAuth(authHeader)

		if !ok || !a.Validate(username, password) {
			log.Printf("[Auth] 认证失败: %s (用户: %s)", req.RemoteAddr, username)
			return req, RequireAuth()
		}

		// 认证成功，移除认证头（不转发给目标服务器）
		req.Header.Del("Proxy-Authorization")
		return req, nil
	})

	// 处理 CONNECT 请求（HTTPS）
	proxy.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		if !a.IsEnabled() {
			return goproxy.OkConnect, host
		}

		authHeader := ctx.Req.Header.Get("Proxy-Authorization")
		username, password, ok := ParseBasicAuth(authHeader)

		if !ok || !a.Validate(username, password) {
			log.Printf("[Auth] CONNECT认证失败: %s -> %s", ctx.Req.RemoteAddr, host)
			return goproxy.RejectConnect, host
		}

		return goproxy.OkConnect, host
	})

	_ = originalHandler // 避免编译警告
}
