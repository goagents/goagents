package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Manager struct {
	agents    map[string]*Agent
	mu        sync.RWMutex
	logger    *zap.Logger
	events    chan Event
	idleTimer *time.Timer
}

func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		agents: make(map[string]*Agent),
		logger: logger,
		events: make(chan Event, 100),
	}
}

func (m *Manager) CreateAgent(config *AgentConfig) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	id := generateAgentID()
	ctx, cancel := context.WithCancel(context.Background())
	
	agent := &Agent{
		ID:           id,
		Name:         id, // Will be set by caller if needed
		Config:       config,
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastActivity: time.Now(),
		ctx:          ctx,
		cancel:       cancel,
		metrics:      &AgentMetrics{},
	}
	
	m.agents[id] = agent
	m.logger.Info("Agent created", zap.String("id", id), zap.String("name", agent.Name))
	
	return agent, nil
}

func (m *Manager) StartAgent(agentID string) error {
	m.mu.Lock()
	agent, exists := m.agents[agentID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("agent not found: %s", agentID)
	}
	
	if agent.Status != StatusPending && agent.Status != StatusStopped {
		m.mu.Unlock()
		return fmt.Errorf("agent %s is in invalid state for starting: %s", agentID, agent.Status)
	}
	
	agent.Status = StatusStarting
	agent.UpdatedAt = time.Now()
	m.mu.Unlock()
	
	go m.runAgent(agent)
	
	m.publishEvent(Event{
		Type:      EventAgentStarted,
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"name": agent.Name,
		},
	})
	
	return nil
}

func (m *Manager) StopAgent(agentID string) error {
	m.mu.Lock()
	agent, exists := m.agents[agentID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("agent not found: %s", agentID)
	}
	
	if agent.Status == StatusStopped || agent.Status == StatusStopping {
		m.mu.Unlock()
		return nil
	}
	
	agent.Status = StatusStopping
	agent.UpdatedAt = time.Now()
	m.mu.Unlock()
	
	agent.cancel()
	
	m.publishEvent(Event{
		Type:      EventAgentStopped,
		AgentID:   agentID,
		Timestamp: time.Now(),
	})
	
	return nil
}

func (m *Manager) GetAgent(agentID string) (*Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	agent, exists := m.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	
	return agent, nil
}

func (m *Manager) ListAgents() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	agents := make([]*Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	
	return agents
}

func (m *Manager) DeleteAgent(agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	agent, exists := m.agents[agentID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	
	if agent.Status == StatusRunning {
		agent.cancel()
	}
	
	delete(m.agents, agentID)
	m.logger.Info("Agent deleted", zap.String("id", agentID))
	
	return nil
}

func (m *Manager) ProcessRequest(agentID string, req *Request) (*Response, error) {
	agent, err := m.GetAgent(agentID)
	if err != nil {
		return nil, err
	}
	
	if agent.Status != StatusRunning {
		if err := m.StartAgent(agentID); err != nil {
			return nil, fmt.Errorf("failed to start agent: %w", err)
		}
		
		timeout := time.NewTimer(30 * time.Second)
		defer timeout.Stop()
		
		for {
			select {
			case <-timeout.C:
				return nil, fmt.Errorf("timeout waiting for agent to start")
			case <-time.After(100 * time.Millisecond):
				agent, _ := m.GetAgent(agentID)
				if agent.Status == StatusRunning {
					goto ready
				}
			}
		}
	}
	
ready:
	agent.mu.Lock()
	agent.LastActivity = time.Now()
	agent.metrics.RequestsTotal++
	agent.mu.Unlock()
	
	m.publishEvent(Event{
		Type:      EventRequestStarted,
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"request_id": req.ID,
		},
	})
	
	resp := &Response{
		ID:      req.ID,
		Content: "Mock response from agent " + agent.Name,
	}
	
	agent.mu.Lock()
	agent.metrics.RequestsSucceeded++
	agent.metrics.LastRequestTime = time.Now()
	agent.mu.Unlock()
	
	m.publishEvent(Event{
		Type:      EventRequestEnded,
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"request_id": req.ID,
			"success":    true,
		},
	})
	
	return resp, nil
}

func (m *Manager) runAgent(agent *Agent) {
	m.logger.Info("Starting agent", zap.String("id", agent.ID), zap.String("name", agent.Name))
	
	agent.mu.Lock()
	agent.Status = StatusRunning
	agent.UpdatedAt = time.Now()
	agent.mu.Unlock()
	
	idleTimeout := 5 * time.Minute
	if agent.Config.Resources.Timeout > 0 {
		idleTimeout = agent.Config.Resources.Timeout
	}
	
	idleTimer := time.NewTimer(idleTimeout)
	defer idleTimer.Stop()
	
	for {
		select {
		case <-agent.ctx.Done():
			m.logger.Info("Agent stopping", zap.String("id", agent.ID))
			agent.mu.Lock()
			agent.Status = StatusStopped
			agent.UpdatedAt = time.Now()
			agent.mu.Unlock()
			return
			
		case <-idleTimer.C:
			agent.mu.Lock()
			lastActivity := agent.LastActivity
			agent.mu.Unlock()
			
			if time.Since(lastActivity) >= idleTimeout {
				m.logger.Info("Agent going idle", zap.String("id", agent.ID))
				agent.mu.Lock()
				agent.Status = StatusIdle
				agent.UpdatedAt = time.Now()
				agent.mu.Unlock()
				
				m.publishEvent(Event{
					Type:      EventAgentIdle,
					AgentID:   agent.ID,
					Timestamp: time.Now(),
				})
			}
			
			idleTimer.Reset(idleTimeout)
		}
	}
}

func (m *Manager) publishEvent(event Event) {
	select {
	case m.events <- event:
	default:
		m.logger.Warn("Event channel full, dropping event", zap.String("type", string(event.Type)))
	}
}

func (m *Manager) Events() <-chan Event {
	return m.events
}

func generateAgentID() string {
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}

func (a *Agent) GetStatus() Status {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Status
}

func (a *Agent) GetMetrics() *AgentMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	metrics := *a.metrics
	return &metrics
}

func (a *Agent) UpdateLastActivity() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.LastActivity = time.Now()
}