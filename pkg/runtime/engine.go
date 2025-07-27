package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goagents/goagents/pkg/agent"
	"github.com/goagents/goagents/pkg/config"
	"github.com/goagents/goagents/pkg/providers"
	"github.com/goagents/goagents/pkg/tools"
	"go.uber.org/zap"
)

type Engine struct {
	config          *config.Config
	agentManager    *agent.Manager
	providerManager *providers.Manager
	toolManager     *tools.Manager
	clusters        map[string]*Cluster
	logger          *zap.Logger
	metrics         *Metrics
	mu              sync.RWMutex
}

type Cluster struct {
	Name      string
	Config    *config.AgentCluster
	Agents    map[string]*agent.Agent
	Status    ClusterStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	mu        sync.RWMutex
}

type ClusterStatus string

const (
	ClusterStatusPending ClusterStatus = "pending"
	ClusterStatusRunning ClusterStatus = "running"
	ClusterStatusStopped ClusterStatus = "stopped"
	ClusterStatusFailed  ClusterStatus = "failed"
)

type Metrics struct {
	ClustersTotal      int64
	AgentsTotal        int64
	RequestsTotal      int64
	RequestsSucceeded  int64
	RequestsFailed     int64
	AverageResponseTime time.Duration
	mu                 sync.RWMutex
}

func NewEngine(cfg *config.Config, logger *zap.Logger) (*Engine, error) {
	engine := &Engine{
		config:          cfg,
		agentManager:    agent.NewManager(logger),
		providerManager: providers.NewManager(),
		toolManager:     tools.NewManager(),
		clusters:        make(map[string]*Cluster),
		logger:          logger,
		metrics:         &Metrics{},
	}
	
	if err := engine.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}
	
	return engine, nil
}

func (e *Engine) initializeProviders() error {
	// Initialize Anthropic provider
	if e.config.Providers.Anthropic != nil {
		providerConfig := &providers.AnthropicConfig{
			APIKey:  e.config.Providers.Anthropic.APIKey,
			BaseURL: e.config.Providers.Anthropic.BaseURL,
			Version: e.config.Providers.Anthropic.Version,
		}
		provider := providers.NewAnthropicProvider(providerConfig)
		e.providerManager.RegisterProvider("anthropic", provider)
		e.logger.Info("Registered Anthropic provider")
	}
	
	// Initialize OpenAI provider
	if e.config.Providers.OpenAI != nil {
		providerConfig := &providers.OpenAIConfig{
			APIKey:  e.config.Providers.OpenAI.APIKey,
			BaseURL: e.config.Providers.OpenAI.BaseURL,
			OrgID:   e.config.Providers.OpenAI.OrgID,
		}
		provider := providers.NewOpenAIProvider(providerConfig)
		e.providerManager.RegisterProvider("openai", provider)
		e.logger.Info("Registered OpenAI provider")
	}
	
	// Initialize Gemini provider
	if e.config.Providers.Gemini != nil {
		providerConfig := &providers.GeminiConfig{
			APIKey:    e.config.Providers.Gemini.APIKey,
			ProjectID: e.config.Providers.Gemini.ProjectID,
		}
		provider := providers.NewGeminiProvider(providerConfig)
		e.providerManager.RegisterProvider("gemini", provider)
		e.logger.Info("Registered Gemini provider")
	}
	
	return nil
}

func (e *Engine) DeployCluster(clusterConfig *config.AgentCluster) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	clusterName := clusterConfig.Metadata.Name
	if _, exists := e.clusters[clusterName]; exists {
		return fmt.Errorf("cluster %s already exists", clusterName)
	}
	
	cluster := &Cluster{
		Name:      clusterName,
		Config:    clusterConfig,
		Agents:    make(map[string]*agent.Agent),
		Status:    ClusterStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	e.clusters[clusterName] = cluster
	e.metrics.ClustersTotal++
	
	e.logger.Info("Cluster deployed", zap.String("name", clusterName))
	
	// Start cluster in background
	go e.startCluster(cluster)
	
	return nil
}

func (e *Engine) startCluster(cluster *Cluster) {
	cluster.mu.Lock()
	cluster.Status = ClusterStatusRunning
	cluster.UpdatedAt = time.Now()
	cluster.mu.Unlock()
	
	e.logger.Info("Starting cluster", zap.String("name", cluster.Name))
	
	// Initialize agents for the cluster
	for _, agentConfig := range cluster.Config.Spec.Agents {
		if err := e.createAgent(cluster, &agentConfig); err != nil {
			e.logger.Error("Failed to create agent", 
				zap.String("cluster", cluster.Name),
				zap.String("agent", agentConfig.Name),
				zap.Error(err))
			continue
		}
	}
	
	e.logger.Info("Cluster started", zap.String("name", cluster.Name))
}

func (e *Engine) createAgent(cluster *Cluster, agentConfig *config.Agent) error {
	// Convert config to agent config
	agentCfg := &agent.AgentConfig{
		Provider:     agentConfig.Provider,
		Model:        agentConfig.Model,
		SystemPrompt: agentConfig.SystemPrompt,
		Environment:  agentConfig.Environment,
	}
	
	// Convert tools
	for _, toolConfig := range agentConfig.Tools {
		toolCfg := &tools.Config{
			Type:     toolConfig.Type,
			Name:     toolConfig.Name,
			URL:      toolConfig.URL,
			Endpoint: toolConfig.Endpoint,
			Server:   toolConfig.Server,
			Config:   toolConfig.Config,
		}
		
		if toolConfig.Auth != nil {
			toolCfg.Auth = &tools.AuthConfig{
				Type:   toolConfig.Auth.Type,
				Token:  toolConfig.Auth.Token,
				APIKey: toolConfig.Auth.APIKey,
				Secret: toolConfig.Auth.Secret,
			}
		}
		
		tool, err := tools.CreateTool(toolCfg)
		if err != nil {
			e.logger.Warn("Failed to create tool", 
				zap.String("tool", toolConfig.Name),
				zap.Error(err))
			continue
		}
		
		e.toolManager.RegisterTool(tool)
	}
	
	// Create agent
	newAgent, err := e.agentManager.CreateAgent(agentCfg)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	
	newAgent.Name = agentConfig.Name
	newAgent.ClusterName = cluster.Name
	
	cluster.mu.Lock()
	cluster.Agents[agentConfig.Name] = newAgent
	cluster.mu.Unlock()
	
	e.metrics.AgentsTotal++
	
	e.logger.Info("Agent created", 
		zap.String("cluster", cluster.Name),
		zap.String("agent", agentConfig.Name),
		zap.String("provider", agentConfig.Provider))
	
	return nil
}

func (e *Engine) ProcessRequest(clusterName, agentName string, req *agent.Request) (*agent.Response, error) {
	cluster, err := e.getCluster(clusterName)
	if err != nil {
		return nil, err
	}
	
	cluster.mu.RLock()
	targetAgent, exists := cluster.Agents[agentName]
	cluster.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("agent %s not found in cluster %s", agentName, clusterName)
	}
	
	// Check if provider is available
	provider, exists := e.providerManager.GetProvider(targetAgent.Config.Provider)
	if !exists {
		return nil, fmt.Errorf("provider %s not available", targetAgent.Config.Provider)
	}
	
	start := time.Now()
	e.metrics.mu.Lock()
	e.metrics.RequestsTotal++
	e.metrics.mu.Unlock()
	
	// Convert agent request to provider request
	providerReq := &providers.ChatRequest{
		Model:    targetAgent.Config.Model,
		Messages: make([]providers.Message, len(req.Messages)),
	}
	
	for i, msg := range req.Messages {
		providerReq.Messages[i] = providers.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	
	// Add system prompt if available
	if targetAgent.Config.SystemPrompt != "" {
		systemMsg := providers.Message{
			Role:    "system",
			Content: targetAgent.Config.SystemPrompt,
		}
		providerReq.Messages = append([]providers.Message{systemMsg}, providerReq.Messages...)
	}
	
	ctx := context.Background()
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}
	
	// Call provider
	providerResp, err := provider.Chat(ctx, providerReq)
	if err != nil {
		e.metrics.mu.Lock()
		e.metrics.RequestsFailed++
		e.metrics.mu.Unlock()
		
		return &agent.Response{
			ID:    req.ID,
			Error: fmt.Sprintf("provider error: %v", err),
		}, nil
	}
	
	duration := time.Since(start)
	e.metrics.mu.Lock()
	e.metrics.RequestsSucceeded++
	e.metrics.AverageResponseTime = (e.metrics.AverageResponseTime + duration) / 2
	e.metrics.mu.Unlock()
	
	// Update agent activity
	targetAgent.UpdateLastActivity()
	
	// Convert provider response to agent response
	resp := &agent.Response{
		ID:      req.ID,
		Content: providerResp.Content,
		Metadata: map[string]interface{}{
			"model":    providerResp.Model,
			"provider": targetAgent.Config.Provider,
			"usage":    providerResp.Usage,
		},
	}
	
	return resp, nil
}

func (e *Engine) getCluster(name string) (*Cluster, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	cluster, exists := e.clusters[name]
	if !exists {
		return nil, fmt.Errorf("cluster not found: %s", name)
	}
	
	return cluster, nil
}

func (e *Engine) ListClusters() []*Cluster {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	clusters := make([]*Cluster, 0, len(e.clusters))
	for _, cluster := range e.clusters {
		clusters = append(clusters, cluster)
	}
	
	return clusters
}

func (e *Engine) GetClusterStatus(name string) (*Cluster, error) {
	return e.getCluster(name)
}

func (e *Engine) StopCluster(name string) error {
	cluster, err := e.getCluster(name)
	if err != nil {
		return err
	}
	
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	
	if cluster.Status == ClusterStatusStopped {
		return nil
	}
	
	// Stop all agents in the cluster
	for _, agent := range cluster.Agents {
		if err := e.agentManager.StopAgent(agent.ID); err != nil {
			e.logger.Warn("Failed to stop agent", 
				zap.String("agent", agent.Name),
				zap.Error(err))
		}
	}
	
	cluster.Status = ClusterStatusStopped
	cluster.UpdatedAt = time.Now()
	
	e.logger.Info("Cluster stopped", zap.String("name", name))
	return nil
}

func (e *Engine) DeleteCluster(name string) error {
	if err := e.StopCluster(name); err != nil {
		return err
	}
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	cluster, exists := e.clusters[name]
	if !exists {
		return fmt.Errorf("cluster not found: %s", name)
	}
	
	// Delete all agents
	for _, agent := range cluster.Agents {
		if err := e.agentManager.DeleteAgent(agent.ID); err != nil {
			e.logger.Warn("Failed to delete agent", 
				zap.String("agent", agent.Name),
				zap.Error(err))
		}
	}
	
	delete(e.clusters, name)
	e.metrics.ClustersTotal--
	
	e.logger.Info("Cluster deleted", zap.String("name", name))
	return nil
}

func (e *Engine) GetMetrics() *Metrics {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()
	
	metrics := *e.metrics
	return &metrics
}

func (e *Engine) Close() error {
	e.logger.Info("Shutting down engine")
	
	// Stop all clusters
	for name := range e.clusters {
		if err := e.StopCluster(name); err != nil {
			e.logger.Warn("Failed to stop cluster during shutdown", 
				zap.String("cluster", name),
				zap.Error(err))
		}
	}
	
	// Close providers
	if err := e.providerManager.Close(); err != nil {
		e.logger.Warn("Failed to close providers", zap.Error(err))
	}
	
	// Close tools
	if err := e.toolManager.Close(); err != nil {
		e.logger.Warn("Failed to close tools", zap.Error(err))
	}
	
	return nil
}