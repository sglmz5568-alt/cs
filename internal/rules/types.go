package rules

import (
	"time"
)

type RuleType string

const (
	RuleTypeURLReplace    RuleType = "url_replace"
	RuleTypeHeaderModify  RuleType = "header_modify"
	RuleTypeTokenExtract  RuleType = "token_extract"
	RuleTypeBodyReplace   RuleType = "body_replace"
)

type RuleTarget string

const (
	RuleTargetRequest  RuleTarget = "request"
	RuleTargetResponse RuleTarget = "response"
)

type Rule struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Type      RuleType   `json:"type"`
	Match     string     `json:"match"`
	Replace   string     `json:"replace"`
	Target    RuleTarget `json:"target"`
	Priority  int        `json:"priority"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type TokenRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

type LogEntry struct {
	ID          string            `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Type        string            `json:"type"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	StatusCode  int               `json:"status_code,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Modified    bool              `json:"modified"`
	RulesApplied []string         `json:"rules_applied,omitempty"`
	Error       string            `json:"error,omitempty"`
}
