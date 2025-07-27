package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goagents/goagents/pkg/agent"
	"github.com/goagents/goagents/pkg/config"
	"go.uber.org/zap"
)

// Health and readiness handlers
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

func (s *Server) readyHandler(c *gin.Context) {
	clusters := s.engine.ListClusters()
	runningClusters := 0
	
	for _, cluster := range clusters {
		if cluster.Status == "running" {
			runningClusters++
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":           "ready",
		"clusters_total":   len(clusters),
		"clusters_running": runningClusters,
		"timestamp":        time.Now().UTC(),
	})
}

// Cluster handlers
func (s *Server) listClustersHandler(c *gin.Context) {
	clusters := s.engine.ListClusters()
	
	clusterList := make([]gin.H, len(clusters))
	for i, cluster := range clusters {
		clusterList[i] = gin.H{
			"name":       cluster.Name,
			"status":     cluster.Status,
			"agents":     len(cluster.Agents),
			"created_at": cluster.CreatedAt,
			"updated_at": cluster.UpdatedAt,
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"clusters": clusterList,
		"total":    len(clusters),
	})
}

func (s *Server) createClusterHandler(c *gin.Context) {
	var clusterConfig config.AgentCluster
	if err := c.ShouldBindJSON(&clusterConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid cluster configuration",
			"details": err.Error(),
		})
		return
	}
	
	if err := s.engine.DeployCluster(&clusterConfig); err != nil {
		s.logger.Error("Failed to deploy cluster", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to deploy cluster",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Cluster created successfully",
		"name":    clusterConfig.Metadata.Name,
	})
}

func (s *Server) getClusterHandler(c *gin.Context) {
	clusterName := c.Param("name")
	
	cluster, err := s.engine.GetClusterStatus(clusterName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Cluster not found",
			"details": err.Error(),
		})
		return
	}
	
	agents := make([]gin.H, 0, len(cluster.Agents))
	for _, agent := range cluster.Agents {
		metrics := agent.GetMetrics()
		agents = append(agents, gin.H{
			"id":            agent.ID,
			"name":          agent.Name,
			"status":        agent.GetStatus(),
			"provider":      agent.Config.Provider,
			"model":         agent.Config.Model,
			"created_at":    agent.CreatedAt,
			"updated_at":    agent.UpdatedAt,
			"last_activity": agent.LastActivity,
			"metrics": gin.H{
				"requests_total":     metrics.RequestsTotal,
				"requests_succeeded": metrics.RequestsSucceeded,
				"requests_failed":    metrics.RequestsFailed,
				"response_time":      metrics.ResponseTime,
				"last_request_time":  metrics.LastRequestTime,
			},
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"name":       cluster.Name,
		"status":     cluster.Status,
		"created_at": cluster.CreatedAt,
		"updated_at": cluster.UpdatedAt,
		"agents":     agents,
		"config":     cluster.Config,
	})
}

func (s *Server) deleteClusterHandler(c *gin.Context) {
	clusterName := c.Param("name")
	
	if err := s.engine.DeleteCluster(clusterName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete cluster",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Cluster deleted successfully",
		"name":    clusterName,
	})
}

func (s *Server) scaleClusterHandler(c *gin.Context) {
	clusterName := c.Param("name")
	
	var scaleRequest struct {
		Agent     string `json:"agent" binding:"required"`
		Instances int    `json:"instances" binding:"min=0"`
	}
	
	if err := c.ShouldBindJSON(&scaleRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid scale request",
			"details": err.Error(),
		})
		return
	}
	
	// For demo purposes, simulate scaling operation
	s.logger.Info("Scaling agent",
		zap.String("cluster", clusterName),
		zap.String("agent", scaleRequest.Agent),
		zap.Int("instances", scaleRequest.Instances))
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "Agent scaled successfully",
		"cluster":   clusterName,
		"agent":     scaleRequest.Agent,
		"instances": scaleRequest.Instances,
	})
}

// Agent handlers
func (s *Server) listAgentsHandler(c *gin.Context) {
	clusterFilter := c.Query("cluster")
	
	clusters := s.engine.ListClusters()
	var allAgents []gin.H
	
	for _, cluster := range clusters {
		if clusterFilter != "" && cluster.Name != clusterFilter {
			continue
		}
		
		for _, agent := range cluster.Agents {
			metrics := agent.GetMetrics()
			allAgents = append(allAgents, gin.H{
				"id":            agent.ID,
				"name":          agent.Name,
				"cluster":       agent.ClusterName,
				"status":        agent.GetStatus(),
				"provider":      agent.Config.Provider,
				"model":         agent.Config.Model,
				"created_at":    agent.CreatedAt,
				"updated_at":    agent.UpdatedAt,
				"last_activity": agent.LastActivity,
				"metrics": gin.H{
					"requests_total":     metrics.RequestsTotal,
					"requests_succeeded": metrics.RequestsSucceeded,
					"requests_failed":    metrics.RequestsFailed,
				},
			})
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"agents": allAgents,
		"total":  len(allAgents),
	})
}

func (s *Server) getAgentHandler(c *gin.Context) {
	agentID := c.Param("id")
	
	// Find agent across all clusters
	clusters := s.engine.ListClusters()
	for _, cluster := range clusters {
		for _, agent := range cluster.Agents {
			if agent.ID == agentID {
				metrics := agent.GetMetrics()
				c.JSON(http.StatusOK, gin.H{
					"id":            agent.ID,
					"name":          agent.Name,
					"cluster":       agent.ClusterName,
					"status":        agent.GetStatus(),
					"provider":      agent.Config.Provider,
					"model":         agent.Config.Model,
					"system_prompt": agent.Config.SystemPrompt,
					"created_at":    agent.CreatedAt,
					"updated_at":    agent.UpdatedAt,
					"last_activity": agent.LastActivity,
					"metrics":       metrics,
					"config":        agent.Config,
				})
				return
			}
		}
	}
	
	c.JSON(http.StatusNotFound, gin.H{
		"error": "Agent not found",
	})
}

func (s *Server) chatHandler(c *gin.Context) {
	agentID := c.Param("id")
	
	var chatRequest struct {
		Messages []agent.Message        `json:"messages" binding:"required"`
		Context  map[string]interface{} `json:"context,omitempty"`
		Timeout  int                    `json:"timeout,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&chatRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid chat request",
			"details": err.Error(),
		})
		return
	}
	
	// Find agent's cluster and name
	clusters := s.engine.ListClusters()
	var clusterName, agentName string
	
	for _, cluster := range clusters {
		for _, agent := range cluster.Agents {
			if agent.ID == agentID {
				clusterName = cluster.Name
				agentName = agent.Name
				break
			}
		}
		if clusterName != "" {
			break
		}
	}
	
	if clusterName == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Agent not found",
		})
		return
	}
	
	// Create request
	req := &agent.Request{
		ID:       fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Messages: chatRequest.Messages,
		Context:  chatRequest.Context,
	}
	
	if chatRequest.Timeout > 0 {
		req.Timeout = time.Duration(chatRequest.Timeout) * time.Second
	}
	
	// Process request
	resp, err := s.engine.ProcessRequest(clusterName, agentName, req)
	if err != nil {
		s.logger.Error("Failed to process request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process request",
			"details": err.Error(),
		})
		return
	}
	
	if resp.Error != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": resp.Error,
		})
		return
	}
	
	c.JSON(http.StatusOK, resp)
}

func (s *Server) streamHandler(c *gin.Context) {
	agentID := c.Param("id")
	
	// For demo purposes, simulate streaming
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	
	// Mock streaming response
	chunks := []string{
		"Hello, this is a streaming response",
		" from agent " + agentID + ".",
		" Each chunk is sent separately",
		" to demonstrate real-time streaming capabilities.",
		" This concludes the demo stream.",
	}
	
	for i, chunk := range chunks {
		data := map[string]interface{}{
			"id":      i,
			"delta":   chunk,
			"content": chunk,
			"done":    i == len(chunks)-1,
		}
		
		jsonData, _ := json.Marshal(data)
		c.SSEvent("message", string(jsonData))
		c.Writer.Flush()
		
		time.Sleep(500 * time.Millisecond)
	}
}

// Metrics handler
func (s *Server) metricsHandler(c *gin.Context) {
	metrics := s.engine.GetMetrics()
	
	c.JSON(http.StatusOK, gin.H{
		"clusters_total":        metrics.ClustersTotal,
		"agents_total":          metrics.AgentsTotal,
		"requests_total":        metrics.RequestsTotal,
		"requests_succeeded":    metrics.RequestsSucceeded,
		"requests_failed":       metrics.RequestsFailed,
		"average_response_time": metrics.AverageResponseTime,
		"timestamp":             time.Now().UTC(),
	})
}

// System info handler
func (s *Server) infoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":        "GoAgents",
		"version":     "1.0.0",
		"description": "AI Agent Orchestration Platform",
		"api_version": "v1",
		"endpoints": gin.H{
			"health":    "/health",
			"ready":     "/ready",
			"clusters":  "/api/v1/clusters",
			"agents":    "/api/v1/agents",
			"metrics":   "/api/v1/metrics",
			"prometheus": s.config.Server.Metrics.Path,
		},
		"features": []string{
			"multi-provider-support",
			"dynamic-scaling",
			"tool-connectivity",
			"streaming-responses",
			"metrics-collection",
		},
		"providers": []string{"anthropic", "openai", "gemini"},
		"tools":     []string{"http", "websocket", "mcp"},
	})
}