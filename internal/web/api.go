package web

import (
	"encoding/json"
	"net/http"
	"time"

	"sunnyproxy/internal/proxy"
	"sunnyproxy/internal/rules"
)

type API struct {
	engine  *rules.Engine
	wrapper *proxy.Wrapper
}

func NewAPI(engine *rules.Engine, wrapper *proxy.Wrapper) *API {
	return &API{
		engine:  engine,
		wrapper: wrapper,
	}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/rules", a.handleRules)
	mux.HandleFunc("/api/rules/", a.handleRule)
	mux.HandleFunc("/api/tokens", a.handleTokens)
	mux.HandleFunc("/api/status", a.handleStatus)
	mux.HandleFunc("/ssl", a.handleCertDownload)
	mux.HandleFunc("/proxy.pac", a.handlePAC)
}

func (a *API) handleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Token")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rules := a.engine.GetRules()
		json.NewEncoder(w).Encode(rules)

	case http.MethodPost:
		var rule rules.Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if err := a.engine.AddRule(rule); err != nil {
			http.Error(w, `{"error":"Failed to add rule"}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (a *API) handleRule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Token")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	id := r.URL.Path[len("/api/rules/"):]
	if id == "" {
		http.Error(w, `{"error":"Rule ID required"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var rule rules.Rule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if err := a.engine.UpdateRule(id, rule); err != nil {
			http.Error(w, `{"error":"Failed to update rule"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	case http.MethodDelete:
		if err := a.engine.DeleteRule(id); err != nil {
			http.Error(w, `{"error":"Failed to delete rule"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (a *API) handleTokens(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	tokens := a.engine.GetTokens()
	json.NewEncoder(w).Encode(tokens)
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	status := map[string]interface{}{
		"status":     "running",
		"proxy_port": a.wrapper.GetPort(),
		"uptime":     time.Now().Format(time.RFC3339),
		"rules":      len(a.engine.GetRules()),
		"tokens":     len(a.engine.GetTokens()),
	}
	json.NewEncoder(w).Encode(status)
}

func (a *API) handleCertDownload(w http.ResponseWriter, r *http.Request) {
	cert := a.wrapper.ExportCert()
	if cert == nil || len(cert) == 0 {
		http.Error(w, "Certificate not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", "attachment; filename=SunnyProxy.cer")
	w.Write(cert)
}

func (a *API) handlePAC(w http.ResponseWriter, r *http.Request) {
	// 获取代理地址（从请求头或使用默认值）
	proxyHost := r.URL.Query().Get("host")
	proxyPort := r.URL.Query().Get("port")

	if proxyHost == "" {
		proxyHost = "centerbeam.proxy.rlwy.net"
	}
	if proxyPort == "" {
		proxyPort = "12964"
	}

	// 从规则中提取需要代理的域名
	rules := a.engine.GetRules()
	var domains []string
	for _, rule := range rules {
		if rule.Enabled && rule.Match != "" {
			// 提取域名部分
			domains = append(domains, rule.Match)
		}
	}

	// 生成 PAC 脚本
	pac := `function FindProxyForURL(url, host) {
    // 需要走代理的域名/关键词
    var proxyPatterns = [`

	for i, domain := range domains {
		if i > 0 {
			pac += ","
		}
		pac += `"` + domain + `"`
	}

	pac += `];

    // 检查是否匹配
    for (var i = 0; i < proxyPatterns.length; i++) {
        if (shExpMatch(host, "*" + proxyPatterns[i] + "*") ||
            url.indexOf(proxyPatterns[i]) !== -1) {
            return "PROXY ` + proxyHost + `:` + proxyPort + `";
        }
    }

    // 其他请求直连
    return "DIRECT";
}`

	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(pac))
}
