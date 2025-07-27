package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Loader struct {
	viper *viper.Viper
}

func NewLoader() *Loader {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("GOAGENTS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	
	setDefaults(v)
	
	return &Loader{viper: v}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.timeout", "30s")
	v.SetDefault("server.log_level", "info")
	v.SetDefault("server.metrics.enabled", true)
	v.SetDefault("server.metrics.path", "/metrics")
	v.SetDefault("server.metrics.port", 9090)
}

func (l *Loader) LoadConfig(configPath string) (*Config, error) {
	if configPath != "" {
		if err := l.loadFromFile(configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file %s: %w", configPath, err)
		}
	}
	
	var config Config
	if err := l.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	if err := l.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &config, nil
}

func (l *Loader) LoadAgentCluster(clusterPath string) (*AgentCluster, error) {
	data, err := os.ReadFile(clusterPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cluster file %s: %w", clusterPath, err)
	}
	
	var cluster AgentCluster
	ext := strings.ToLower(filepath.Ext(clusterPath))
	
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cluster); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &cluster); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
	
	if err := l.validateAgentCluster(&cluster); err != nil {
		return nil, fmt.Errorf("cluster validation failed: %w", err)
	}
	
	return &cluster, nil
}

func (l *Loader) loadFromFile(configPath string) error {
	l.viper.SetConfigFile(configPath)
	
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return fmt.Errorf("config file not found: %s", configPath)
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	return nil
}

func (l *Loader) validateConfig(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	if config.Server.Metrics.Enabled && (config.Server.Metrics.Port <= 0 || config.Server.Metrics.Port > 65535) {
		return fmt.Errorf("invalid metrics port: %d", config.Server.Metrics.Port)
	}
	
	for i, cluster := range config.Clusters {
		if err := l.validateAgentCluster(&cluster); err != nil {
			return fmt.Errorf("cluster %d validation failed: %w", i, err)
		}
	}
	
	return nil
}

func (l *Loader) validateAgentCluster(cluster *AgentCluster) error {
	if cluster.APIVersion == "" {
		cluster.APIVersion = "goagents.dev/v1"
	}
	
	if cluster.Kind == "" {
		cluster.Kind = "AgentCluster"
	}
	
	if cluster.Metadata.Name == "" {
		return fmt.Errorf("cluster name is required")
	}
	
	if cluster.Metadata.Namespace == "" {
		cluster.Metadata.Namespace = "default"
	}
	
	if len(cluster.Spec.Agents) == 0 {
		return fmt.Errorf("at least one agent is required")
	}
	
	agentNames := make(map[string]bool)
	for i, agent := range cluster.Spec.Agents {
		if agent.Name == "" {
			return fmt.Errorf("agent %d: name is required", i)
		}
		
		if agentNames[agent.Name] {
			return fmt.Errorf("duplicate agent name: %s", agent.Name)
		}
		agentNames[agent.Name] = true
		
		if agent.Provider == "" {
			return fmt.Errorf("agent %s: provider is required", agent.Name)
		}
		
		if agent.Model == "" {
			return fmt.Errorf("agent %s: model is required", agent.Name)
		}
		
		if !isValidProvider(agent.Provider) {
			return fmt.Errorf("agent %s: unsupported provider %s", agent.Name, agent.Provider)
		}
		
		for _, dep := range agent.DependsOn {
			if !agentNames[dep] && dep != agent.Name {
				return fmt.Errorf("agent %s: dependency %s not found", agent.Name, dep)
			}
		}
	}
	
	return nil
}

func isValidProvider(provider string) bool {
	validProviders := map[string]bool{
		"anthropic": true,
		"openai":    true,
		"gemini":    true,
	}
	return validProviders[provider]
}

func (l *Loader) WatchConfig(configPath string, callback func(*Config)) error {
	l.viper.SetConfigFile(configPath)
	l.viper.WatchConfig()
	l.viper.OnConfigChange(func(e fsnotify.Event) {
		config, err := l.LoadConfig(configPath)
		if err != nil {
			return
		}
		callback(config)
	})
	return nil
}