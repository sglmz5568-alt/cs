package proxy

import (
	"net/http"
	"strings"

	"github.com/elazarl/goproxy"
	"sunnyproxy/internal/logger"
	"sunnyproxy/internal/rules"
)

type Handler struct {
	modifier    *Modifier
	engine      *rules.Engine
	broadcaster *logger.Broadcaster
}

func NewHandler(engine *rules.Engine) *Handler {
	return &Handler{
		modifier:    NewModifier(engine),
		engine:      engine,
		broadcaster: logger.GetBroadcaster(),
	}
}

func (h *Handler) SetupHandlers(proxy *goproxy.ProxyHttpServer) {
	proxy.OnRequest().DoFunc(h.handleRequest)
	proxy.OnResponse().DoFunc(h.handleResponse)
}

func (h *Handler) handleRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	url := req.URL.String()
	if req.URL.Scheme == "" {
		url = "https://" + req.Host + req.URL.Path
		if req.URL.RawQuery != "" {
			url += "?" + req.URL.RawQuery
		}
	}
	method := req.Method

	headers := make(map[string]string)
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	h.modifier.ExtractTokens(url, headers)

	newURL, appliedRules, modified := h.modifier.ModifyURL(url)
	if modified {
		if strings.HasPrefix(newURL, "https://") {
			newURL = newURL[8:]
		} else if strings.HasPrefix(newURL, "http://") {
			newURL = newURL[7:]
		}

		parts := strings.SplitN(newURL, "/", 2)
		if len(parts) >= 1 {
			req.Host = parts[0]
			req.URL.Host = parts[0]
			req.Header.Set("Host", parts[0])
		}
		if len(parts) >= 2 {
			pathAndQuery := "/" + parts[1]
			if idx := strings.Index(pathAndQuery, "?"); idx != -1 {
				req.URL.Path = pathAndQuery[:idx]
				req.URL.RawQuery = pathAndQuery[idx+1:]
			} else {
				req.URL.Path = pathAndQuery
			}
		}
	}

	h.broadcaster.LogRequest(method, url, modified, appliedRules)

	return req, nil
}

func (h *Handler) handleResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil || ctx.Req == nil {
		return resp
	}

	url := ctx.Req.URL.String()
	if ctx.Req.URL.Scheme == "" {
		url = "https://" + ctx.Req.Host + ctx.Req.URL.Path
	}
	method := ctx.Req.Method
	statusCode := resp.StatusCode

	h.broadcaster.LogResponse(method, url, statusCode)

	return resp
}
