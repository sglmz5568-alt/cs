package proxy

import (
	"net/http"
	"path/filepath"
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

	// 只处理 qmai 相关请求
	if !strings.Contains(url, "qmai") {
		return req, nil
	}

	// 只在 payment-another-info 请求时处理
	if strings.Contains(url, "payment-another-info") {
		// 提取 Token
		headers := make(map[string]string)
		for k, v := range req.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		h.modifier.ExtractTokens(url, headers)

		// 替换 URL: payment-another-info -> payment-info
		newURL := strings.ReplaceAll(url, "payment-another-info", "payment-info")

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

		h.broadcaster.LogRequest(method, url, true, []string{"payment-replace"})
		return req, nil
	}

	// 其他 qmai 请求直接透传，不做任何处理
	return req, nil
}

// getContentType 根据文件扩展名返回正确的 Content-Type
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return ""
	}
}

func (h *Handler) handleResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil || ctx.Req == nil {
		return resp
	}

	// 修正 Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || contentType == "text/plain; charset=utf-8" || contentType == "text/plain" {
		path := ctx.Req.URL.Path
		if idx := strings.Index(path, "?"); idx != -1 {
			path = path[:idx]
		}
		correctType := getContentType(path)
		if correctType != "" {
			resp.Header.Set("Content-Type", correctType)
		} else if contentType == "" {
			resp.Header.Del("Content-Type")
		}
	}

	return resp
}
