package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goagents/goagents/pkg/config"
	"github.com/goagents/goagents/pkg/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Server struct {
	config *config.Config
	engine *runtime.Engine
	logger *zap.Logger
	router *gin.Engine
	server *http.Server
}

func NewServer(cfg *config.Config, engine *runtime.Engine, logger *zap.Logger) *Server {
	// Set Gin mode based on log level
	if cfg.Server.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	
	router := gin.New()
	
	s := &Server{
		config: cfg,
		engine: engine,
		logger: logger,
		router: router,
	}
	
	s.setupRoutes()
	s.setupMiddleware()
	
	return s
}

func (s *Server) setupMiddleware() {
	// Logging middleware
	s.router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
				param.ClientIP,
				param.TimeStamp.Format(time.RFC3339),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
				param.ErrorMessage,
			)
		},
		Output: gin.DefaultWriter,
	}))
	
	// Recovery middleware
	s.router.Use(gin.Recovery())
	
	// CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/ready", s.readyHandler)
	
	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Cluster management
		clusters := v1.Group("/clusters")
		{
			clusters.GET("", s.listClustersHandler)
			clusters.POST("", s.createClusterHandler)
			clusters.GET("/:name", s.getClusterHandler)
			clusters.DELETE("/:name", s.deleteClusterHandler)
			clusters.POST("/:name/scale", s.scaleClusterHandler)
		}
		
		// Agent management
		agents := v1.Group("/agents")
		{
			agents.GET("", s.listAgentsHandler)
			agents.GET("/:id", s.getAgentHandler)
			agents.POST("/:id/chat", s.chatHandler)
			agents.POST("/:id/stream", s.streamHandler)
		}
		
		// Metrics
		v1.GET("/metrics", s.metricsHandler)
		
		// System info
		v1.GET("/info", s.infoHandler)
	}
	
	// Metrics endpoint for Prometheus
	if s.config.Server.Metrics.Enabled {
		s.router.GET(s.config.Server.Metrics.Path, gin.WrapH(promhttp.Handler()))
	}
}

func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.Timeout,
		WriteTimeout: s.config.Server.Timeout,
		IdleTimeout:  120 * time.Second,
	}
	
	s.logger.Info("Starting HTTP server", zap.String("addr", addr))
	
	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	
	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down HTTP server")
		
		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Failed to shutdown server gracefully", zap.Error(err))
			return err
		}
		
		s.logger.Info("HTTP server stopped")
		return nil
		
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}
}