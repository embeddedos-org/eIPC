// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

// IntentEvent represents a decoded neural intent from ENI.
type IntentEvent struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
	SessionID  string  `json:"session_id"`
}

// FeatureStreamEvent carries real-time feature data from ENI providers.
type FeatureStreamEvent struct {
	Features map[string]interface{} `json:"features"`
}

// ToolRequestEvent requests execution of a tool in EAI.
type ToolRequestEvent struct {
	Tool       string                 `json:"tool"`
	Args       map[string]interface{} `json:"args"`
	Permission string                 `json:"permission,omitempty"`
	AuditID    string                 `json:"audit_id,omitempty"`
}

// AckEvent acknowledges processing of a request.
type AckEvent struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

// PolicyResultEvent carries an authorization decision.
type PolicyResultEvent struct {
	RequestID string `json:"request_id"`
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason,omitempty"`
}

// HeartbeatEvent signals service liveness.
type HeartbeatEvent struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

// AuditEvent records a controlled action for auditability.
type AuditEvent struct {
	RequestID string                 `json:"request_id"`
	Actor     string                 `json:"actor"`
	Action    string                 `json:"action"`
	Target    string                 `json:"target"`
	Decision  string                 `json:"decision"`
	Result    string                 `json:"result"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// ChatRequestEvent carries a chat message from ebot to EAI.
type ChatRequestEvent struct {
	SessionID  string `json:"session_id"`
	UserPrompt string `json:"user_prompt"`
	Model      string `json:"model,omitempty"`
	MaxTokens  int    `json:"max_tokens,omitempty"`
}

// ChatResponseEvent carries a chat response from EAI back to ebot.
type ChatResponseEvent struct {
	SessionID  string `json:"session_id"`
	Response   string `json:"response"`
	Model      string `json:"model,omitempty"`
	TokensUsed int    `json:"tokens_used,omitempty"`
}

// CompleteRequestEvent carries a completion request from ebot to EAI.
type CompleteRequestEvent struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

// CompleteResponseEvent carries a completion response from EAI back to ebot.
type CompleteResponseEvent struct {
	SessionID  string `json:"session_id"`
	Completion string `json:"completion"`
	Model      string `json:"model,omitempty"`
	TokensUsed int    `json:"tokens_used,omitempty"`
}
