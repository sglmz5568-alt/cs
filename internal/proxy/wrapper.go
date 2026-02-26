package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	"sunnyproxy/internal/domainfilter"
	"sunnyproxy/internal/rules"
)

type Wrapper struct {
	mu          sync.RWMutex
	proxy       *goproxy.ProxyHttpServer
	port        int
	caCert      []byte
	caKey       []byte
	certPool    *x509.CertPool
	engine      *rules.Engine
	transport   *http.Transport
	authEnabled bool
	authUser    string
	authPass    string
}

func NewWrapper() *Wrapper {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	// 创建优化的 Transport，启用连接池和 Keep-Alive
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// 设置代理的 Transport
	proxy.Tr = transport

	return &Wrapper{
		proxy:     proxy,
		transport: transport,
	}
}

func (w *Wrapper) SetEngine(engine *rules.Engine) *Wrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.engine = engine
	return w
}

func (w *Wrapper) SetPort(port int) *Wrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.port = port
	return w
}

func (w *Wrapper) GetProxy() *goproxy.ProxyHttpServer {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.proxy
}

func (w *Wrapper) GetPort() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.port
}

func (w *Wrapper) Start(addr string) error {
	w.mu.Lock()
	proxy := w.proxy
	w.mu.Unlock()

	return http.ListenAndServe(addr, proxy)
}

func (w *Wrapper) Stop() {
	// 关闭连接池
	if w.transport != nil {
		w.transport.CloseIdleConnections()
	}
}

func (w *Wrapper) SetCA(cert, key []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.caCert = cert
	w.caKey = key

	goproxyCa, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}

	if goproxyCa.Leaf, err = x509.ParseCertificate(goproxyCa.Certificate[0]); err != nil {
		return err
	}

	goproxy.GoproxyCa = goproxyCa
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}

	return nil
}

func (w *Wrapper) ExportCert() []byte {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.caCert
}

// shouldMitm 检查是否需要对该域名进行 MITM
func (w *Wrapper) shouldMitm(host string) bool {
	// 只对包含 qmai.cn 的域名进行 MITM
	if strings.Contains(host, "qmai.cn") {
		return true
	}
	return false
}

func (w *Wrapper) EnableMITM() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 获取域名过滤器
	domainFilter := domainfilter.GetFilter()

	// 设置代理认证（使用 goproxy 内置的认证机制）
	if w.authEnabled {
		auth.ProxyBasic(w.proxy, "SunnyProxy", func(user, passwd string) bool {
			if user == w.authUser && passwd == w.authPass {
				return true
			}
			log.Printf("[Auth] 认证失败，用户名: %s", user)
			return false
		})
		log.Printf("[Auth] 代理认证已启用")
	}

	// HTTPS 请求处理（CONNECT方法）
	w.proxy.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		// 检查域名白名单
		if !domainFilter.IsAllowed(host) {
			log.Printf("[DomainFilter] 拒绝访问: %s", host)
			return goproxy.RejectConnect, host
		}

		if w.shouldMitm(host) {
			// 需要处理的域名：进行 MITM 解密
			return goproxy.MitmConnect, host
		}
		// 其他域名：直接透传，不解密
		return goproxy.OkConnect, host
	})

	// HTTP 请求处理
	w.proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		host := req.Host
		if host == "" {
			host = req.URL.Host
		}

		// 检查域名白名单
		if !domainFilter.IsAllowed(host) {
			log.Printf("[DomainFilter] 拒绝HTTP访问: %s", host)
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "Access Denied")
		}

		return req, nil
	})
}

// SetAuth 设置代理认证
func (w *Wrapper) SetAuth(username, password string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.authUser = username
	w.authPass = password
	w.authEnabled = true
}
