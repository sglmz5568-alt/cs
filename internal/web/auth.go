package web

import (
	"net/http"
	"strings"

	"sunnyproxy/pkg/config"
)

type AuthMiddleware struct {
	config *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{config: cfg}
}

func (a *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.config.Security.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/ssl" || r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("X-API-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token != a.config.Security.APIToken {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if !a.isIPAllowed(r) {
			http.Error(w, `{"error":"IP not allowed"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *AuthMiddleware) isIPAllowed(r *http.Request) bool {
	if len(a.config.Security.AllowedIPs) == 0 {
		return true
	}

	for _, allowed := range a.config.Security.AllowedIPs {
		if allowed == "0.0.0.0/0" || allowed == "*" {
			return true
		}
	}

	clientIP := r.RemoteAddr
	if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
		clientIP = clientIP[:idx]
	}

	for _, allowed := range a.config.Security.AllowedIPs {
		if allowed == clientIP {
			return true
		}
	}

	return false
}
