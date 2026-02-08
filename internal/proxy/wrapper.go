package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	"sunnyproxy/internal/rules"
)

type Wrapper struct {
	mu        sync.RWMutex
	proxy     *goproxy.ProxyHttpServer
	port      int
	caCert    []byte
	caKey     []byte
	certPool  *x509.CertPool
	engine    *rules.Engine
	transport *http.Transport
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

	// 只对目标域名进行 MITM，其他直接透传
	w.proxy.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		if w.shouldMitm(host) {
			// 需要处理的域名：进行 MITM 解密
			return goproxy.MitmConnect, host
		}
		// 其他域名：直接透传，不解密
		return goproxy.OkConnect, host
	})
}
