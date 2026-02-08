package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"sync"

	"github.com/elazarl/goproxy"
)

type Wrapper struct {
	mu       sync.RWMutex
	proxy    *goproxy.ProxyHttpServer
	port     int
	caCert   []byte
	caKey    []byte
	certPool *x509.CertPool
}

func NewWrapper() *Wrapper {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	return &Wrapper{
		proxy: proxy,
	}
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

func (w *Wrapper) EnableMITM() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
}
