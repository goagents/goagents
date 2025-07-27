package config

import (
	"time"
)

type AgentCluster struct {
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Kind       string            `yaml:"kind" json:"kind"`
	Metadata   Metadata          `yaml:"metadata" json:"metadata"`
	Spec       AgentClusterSpec  `yaml:"spec" json:"spec"`
}

type Metadata struct {
	Name      string            `yaml:"name" json:"name"`
	Namespace string            `yaml:"namespace" json:"namespace"`
	Labels    map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type AgentClusterSpec struct {
	ResourcePolicy ResourcePolicy `yaml:"resource_policy" json:"resource_policy"`
	Agents         []Agent        `yaml:"agents" json:"agents"`
}

type ResourcePolicy struct {
	MaxConcurrentAgents int           `yaml:"max_concurrent_agents" json:"max_concurrent_agents"`
	IdleTimeout         time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	ScaleToZero         bool          `yaml:"scale_to_zero" json:"scale_to_zero"`
}

type Agent struct {
	Name         string            `yaml:"name" json:"name"`
	Provider     string            `yaml:"provider" json:"provider"`
	Model        string            `yaml:"model" json:"model"`
	SystemPrompt string            `yaml:"system_prompt,omitempty" json:"system_prompt,omitempty"`
	Tools        []Tool            `yaml:"tools,omitempty" json:"tools,omitempty"`
	Resources    Resources         `yaml:"resources,omitempty" json:"resources,omitempty"`
	Scaling      Scaling           `yaml:"scaling,omitempty" json:"scaling,omitempty"`
	DependsOn    []string          `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Environment  map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
}

type Tool struct {
	Type     string            `yaml:"type" json:"type"`
	Name     string            `yaml:"name" json:"name"`
	URL      string            `yaml:"url,omitempty" json:"url,omitempty"`
	Endpoint string            `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	Server   string            `yaml:"server,omitempty" json:"server,omitempty"`
	Auth     *AuthConfig       `yaml:"auth,omitempty" json:"auth,omitempty"`
	Config   map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

type AuthConfig struct {
	Type   string `yaml:"type" json:"type"`
	Token  string `yaml:"token,omitempty" json:"token,omitempty"`
	APIKey string `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	Secret string `yaml:"secret,omitempty" json:"secret,omitempty"`
}

type Resources struct {
	MemoryLimit string        `yaml:"memory_limit,omitempty" json:"memory_limit,omitempty"`
	CPULimit    string        `yaml:"cpu_limit,omitempty" json:"cpu_limit,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

type Scaling struct {
	MinInstances int `yaml:"min_instances,omitempty" json:"min_instances,omitempty"`
	MaxInstances int `yaml:"max_instances,omitempty" json:"max_instances,omitempty"`
}

type ServerConfig struct {
	Host     string        `yaml:"host" json:"host"`
	Port     int           `yaml:"port" json:"port"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
	LogLevel string        `yaml:"log_level" json:"log_level"`
	Metrics  MetricsConfig `yaml:"metrics" json:"metrics"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path"`
	Port    int    `yaml:"port" json:"port"`
}

type ProviderConfig struct {
	Anthropic *AnthropicConfig `yaml:"anthropic,omitempty" json:"anthropic,omitempty"`
	OpenAI    *OpenAIConfig    `yaml:"openai,omitempty" json:"openai,omitempty"`
	Gemini    *GeminiConfig    `yaml:"gemini,omitempty" json:"gemini,omitempty"`
}

type AnthropicConfig struct {
	APIKey  string `yaml:"api_key" json:"api_key"`
	BaseURL string `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

type OpenAIConfig struct {
	APIKey  string `yaml:"api_key" json:"api_key"`
	BaseURL string `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	OrgID   string `yaml:"org_id,omitempty" json:"org_id,omitempty"`
}

type GeminiConfig struct {
	APIKey    string `yaml:"api_key" json:"api_key"`
	ProjectID string `yaml:"project_id,omitempty" json:"project_id,omitempty"`
}

type Config struct {
	Server    ServerConfig    `yaml:"server" json:"server"`
	Providers ProviderConfig  `yaml:"providers" json:"providers"`
	Clusters  []AgentCluster  `yaml:"clusters" json:"clusters"`
}