package rules

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Engine struct {
	mu      sync.RWMutex
	rules   []Rule
	tokens  []TokenRecord
	storage *Storage
}

func NewEngine(storagePath string) (*Engine, error) {
	storage, err := NewStorage(storagePath)
	if err != nil {
		return nil, err
	}

	rules, err := storage.Load()
	if err != nil {
		rules = DefaultRules()
	}

	return &Engine{
		rules:   rules,
		tokens:  make([]TokenRecord, 0),
		storage: storage,
	}, nil
}

func DefaultRules() []Rule {
	now := time.Now()
	return []Rule{
		{
			ID:        "1",
			Name:      "qmai 域名替换",
			Type:      RuleTypeURLReplace,
			Match:     "miniapp.qmai.cn",
			Replace:   "webapi2.qmai.cn",
			Target:    RuleTargetRequest,
			Priority:  1,
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "2",
			Name:      "payment 路径替换",
			Type:      RuleTypeURLReplace,
			Match:     "payment-another-info",
			Replace:   "payment-info",
			Target:    RuleTargetRequest,
			Priority:  2,
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "3",
			Name:      "提取 Qm-User-Token",
			Type:      RuleTypeTokenExtract,
			Match:     "Qm-User-Token",
			Replace:   "",
			Target:    RuleTargetRequest,
			Priority:  3,
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func (e *Engine) GetRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]Rule, len(e.rules))
	copy(result, e.rules)
	return result
}

func (e *Engine) GetEnabledRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var result []Rule
	for _, r := range e.rules {
		if r.Enabled {
			result = append(result, r)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority < result[j].Priority
	})
	return result
}

func (e *Engine) AddRule(rule Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule.ID == "" {
		rule.ID = uuid.New().String()[:8]
	}
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	e.rules = append(e.rules, rule)
	return e.storage.Save(e.rules)
}

func (e *Engine) UpdateRule(id string, rule Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, r := range e.rules {
		if r.ID == id {
			rule.ID = id
			rule.CreatedAt = r.CreatedAt
			rule.UpdatedAt = time.Now()
			e.rules[i] = rule
			return e.storage.Save(e.rules)
		}
	}
	return nil
}

func (e *Engine) DeleteRule(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, r := range e.rules {
		if r.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			return e.storage.Save(e.rules)
		}
	}
	return nil
}

func (e *Engine) ApplyURLRules(url string) (string, []string) {
	rules := e.GetEnabledRules()
	var applied []string

	for _, r := range rules {
		if r.Type != RuleTypeURLReplace || r.Target != RuleTargetRequest {
			continue
		}
		if strings.Contains(url, r.Match) {
			url = strings.ReplaceAll(url, r.Match, r.Replace)
			applied = append(applied, r.ID)
		}
	}

	return url, applied
}

func (e *Engine) ApplyHeaderRules(headers map[string]string) (map[string]string, []string) {
	rules := e.GetEnabledRules()
	var applied []string

	for _, r := range rules {
		if r.Type != RuleTypeHeaderModify || r.Target != RuleTargetRequest {
			continue
		}
		if _, ok := headers[r.Match]; ok {
			headers[r.Match] = r.Replace
			applied = append(applied, r.ID)
		}
	}

	return headers, applied
}

func (e *Engine) ExtractTokens(url string, headers map[string]string) []TokenRecord {
	rules := e.GetEnabledRules()
	var tokens []TokenRecord

	for _, r := range rules {
		if r.Type != RuleTypeTokenExtract {
			continue
		}
		if value, ok := headers[r.Match]; ok && value != "" {
			token := TokenRecord{
				ID:        uuid.New().String()[:8],
				Name:      r.Match,
				Value:     value,
				URL:       url,
				Timestamp: time.Now(),
			}
			tokens = append(tokens, token)
			e.addToken(token)
		}
	}

	return tokens
}

func (e *Engine) addToken(token TokenRecord) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, t := range e.tokens {
		if t.Name == token.Name && t.Value == token.Value {
			e.tokens[i].Timestamp = token.Timestamp
			return
		}
	}

	e.tokens = append(e.tokens, token)
	// 限制最多保存 50 个 Token，防止内存溢出
	if len(e.tokens) > 50 {
		e.tokens = e.tokens[len(e.tokens)-50:]
	}
}

// CleanupOldTokens 清理超过指定时间的 Token
func (e *Engine) CleanupOldTokens(maxAge time.Duration) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var remaining []TokenRecord
	for _, t := range e.tokens {
		if t.Timestamp.After(cutoff) {
			remaining = append(remaining, t)
		}
	}
	removed := len(e.tokens) - len(remaining)
	e.tokens = remaining
	return removed
}

func (e *Engine) GetTokens() []TokenRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]TokenRecord, len(e.tokens))
	copy(result, e.tokens)
	return result
}

func (e *Engine) MatchesURL(url string, pattern string) bool {
	if strings.HasPrefix(pattern, "regex:") {
		re, err := regexp.Compile(pattern[6:])
		if err != nil {
			return false
		}
		return re.MatchString(url)
	}
	return strings.Contains(url, pattern)
}
