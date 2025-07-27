package agent

import (
	"context"
	"sync"
	"time"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusStarting   Status = "starting"
	StatusRunning    Status = "running"
	StatusIdle       Status = "idle"
	StatusStopping   Status = "stopping"
	StatusStopped    Status = "stopped"
	StatusFailed     Status = "failed"
)

type Agent struct {
	ID           string
	Name         string
	ClusterName  string
	Config       *AgentConfig
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastActivity time.Time
	ErrorMessage string
	
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	metrics   *AgentMetrics
}

type AgentConfig struct {
	Provider     string
	Model        string
	SystemPrompt string
	Tools        []ToolConfig
	Resources    ResourceConfig
	Scaling      ScalingConfig
	Environment  map[string]string
}

type ToolConfig struct {
	Type     string
	Name     string
	URL      string
	Endpoint string
	Server   string
	Auth     *AuthConfig
	Config   map[string]string
}

type AuthConfig struct {
	Type   string
	Token  string
	APIKey string
	Secret string
}

type ResourceConfig struct {
	MemoryLimit string
	CPULimit    string
	Timeout     time.Duration
}

type ScalingConfig struct {
	MinInstances int
	MaxInstances int
}

type AgentMetrics struct {
	RequestsTotal     int64
	RequestsSucceeded int64
	RequestsFailed    int64
	ResponseTime      time.Duration
	MemoryUsage       int64
	CPUUsage          float64
	LastRequestTime   time.Time
}

type Message struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Request struct {
	ID       string                 `json:"id"`
	Messages []Message              `json:"messages"`
	Tools    []string               `json:"tools,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
	Timeout  time.Duration          `json:"timeout,omitempty"`
}

type Response struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	ToolUses []ToolUse              `json:"tool_uses,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ToolUse struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type EventType string

const (
	EventAgentStarted   EventType = "agent.started"
	EventAgentStopped   EventType = "agent.stopped"
	EventAgentFailed    EventType = "agent.failed"
	EventAgentIdle      EventType = "agent.idle"
	EventRequestStarted EventType = "request.started"
	EventRequestEnded   EventType = "request.ended"
)

type Event struct {
	Type      EventType              `json:"type"`
	AgentID   string                 `json:"agent_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}