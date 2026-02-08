package proxy

import (
	"strings"

	"sunnyproxy/internal/logger"
	"sunnyproxy/internal/rules"
)

type Modifier struct {
	engine      *rules.Engine
	broadcaster *logger.Broadcaster
}

func NewModifier(engine *rules.Engine) *Modifier {
	return &Modifier{
		engine:      engine,
		broadcaster: logger.GetBroadcaster(),
	}
}

func (m *Modifier) ModifyURL(url string) (string, []string, bool) {
	newURL, applied := m.engine.ApplyURLRules(url)
	return newURL, applied, len(applied) > 0
}

func (m *Modifier) ModifyHeaders(headers map[string]string) (map[string]string, []string, bool) {
	newHeaders, applied := m.engine.ApplyHeaderRules(headers)
	return newHeaders, applied, len(applied) > 0
}

func (m *Modifier) ExtractTokens(url string, headers map[string]string) []rules.TokenRecord {
	tokens := m.engine.ExtractTokens(url, headers)
	for _, t := range tokens {
		m.broadcaster.LogToken(t.Name, t.Value, url)
	}
	return tokens
}

func (m *Modifier) ShouldModify(url string) bool {
	enabledRules := m.engine.GetEnabledRules()
	for _, r := range enabledRules {
		if r.Type == rules.RuleTypeURLReplace && r.Target == rules.RuleTargetRequest {
			if strings.Contains(url, r.Match) {
				return true
			}
		}
	}
	return false
}

func (m *Modifier) GetAppliedHostFromURL(url string) string {
	newURL, _ := m.engine.ApplyURLRules(url)
	parts := strings.Split(newURL, "/")
	if len(parts) >= 3 {
		hostPart := parts[2]
		if idx := strings.Index(hostPart, ":"); idx != -1 {
			return hostPart[:idx]
		}
		return hostPart
	}
	return ""
}
