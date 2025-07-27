# GoAgents

A production-ready AI agent orchestration platform built with Go, designed for dynamic agent management, multi-provider support, and seamless tool connectivity.

## Features

- **Dynamic Agent Management**: Spin up/down agents on-demand to optimize resource usage
- **Multi-Provider Support**: Built-in support for Anthropic (Claude), OpenAI (GPT), and Google Gemini
- **Tool Connectivity**: Supports MCP (Model Context Protocol), HTTP APIs, and WebSocket connections
- **Production Ready**: Designed for FaaS, Cloud Run, and resource-constrained environments
- **High Performance**: Fast startup, low memory footprint, high concurrency with Go
- **Configuration**: Support for both YAML and JSON configuration files
- **Observability**: Built-in metrics, logging, and health checks
- **RESTful API**: Complete HTTP API for cluster and agent management
- **CLI Interface**: Comprehensive command-line interface for operations

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/goagents/goagents.git
cd goagents

# Build the binary
go build -o goagents cmd/goagents/main.go

# Or install globally
go install github.com/goagents/goagents/cmd/goagents@latest
```

### Configuration

Create a configuration file with your LLM provider credentials:

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  log_level: info

providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
  openai:
    api_key: "${OPENAI_API_KEY}"
  gemini:
    api_key: "${GOOGLE_API_KEY}"
```

Set your environment variables:

```bash
export ANTHROPIC_API_KEY="your-anthropic-key"
export OPENAI_API_KEY="your-openai-key"
export GOOGLE_API_KEY="your-google-key"
```

### Running GoAgents

Start the server:

```bash
# Start server only
goagents run --config config.yaml

# Start server and deploy a cluster
goagents run --config config.yaml --cluster examples/customer-support.yaml
```

### Deploy an Agent Cluster

```bash
# Deploy from configuration file
goagents deploy --cluster examples/customer-support.yaml

# Validate configuration without deploying
goagents deploy --cluster examples/data-analysis.json --dry-run
```

### Check Status

```bash
# List all clusters
goagents status

# Check specific cluster
goagents status --cluster customer-support

# JSON output
goagents status --output json
```

### Scale Agents

```bash
# Scale an agent to 3 instances
goagents scale --cluster customer-support --agent intent-classifier --instances 3

# Scale down to zero (stop agent)
goagents scale --cluster customer-support --agent response-generator --instances 0
```

### View Logs

```bash
# Show logs for a cluster
goagents logs --cluster customer-support

# Follow logs for specific agent
goagents logs --cluster customer-support --agent intent-classifier --follow

# Show last 100 lines
goagents logs --cluster data-analysis --lines 100
```

## Configuration Reference

### Server Configuration

```yaml
server:
  host: "0.0.0.0"          # Server bind address
  port: 8080               # Server port
  timeout: 30s             # Request timeout
  log_level: info          # Log level (debug, info, warn, error)
  metrics:
    enabled: true          # Enable Prometheus metrics
    path: /metrics         # Metrics endpoint path
    port: 9090            # Metrics server port
```

### Provider Configuration

```yaml
providers:
  anthropic:
    api_key: "sk-ant-..."       # Anthropic API key
    base_url: "https://..."     # Optional: Custom base URL
    version: "2023-06-01"       # API version
    timeout: 60s               # Request timeout
    
  openai:
    api_key: "sk-..."          # OpenAI API key
    base_url: "https://..."     # Optional: Custom base URL
    org_id: "org-..."          # Optional: Organization ID
    timeout: 60s               # Request timeout
    
  gemini:
    api_key: "AIza..."         # Google API key
    project_id: "my-project"   # Optional: GCP Project ID
    timeout: 60s               # Request timeout
```

### Agent Cluster Configuration

```yaml
apiVersion: goagents.dev/v1
kind: AgentCluster
metadata:
  name: my-cluster
  namespace: default
  labels:
    environment: production
spec:
  resource_policy:
    max_concurrent_agents: 5    # Maximum concurrent agents
    idle_timeout: 300s          # Time before agent goes idle
    scale_to_zero: true         # Allow scaling to zero instances
  agents:
    - name: my-agent
      provider: anthropic       # Provider: anthropic, openai, gemini
      model: claude-sonnet-4    # Model name
      system_prompt: "..."      # System prompt
      tools:                    # Optional: Tool configurations
        - type: http            # Tool type: http, websocket, mcp
          name: api_tool
          url: "https://api.example.com"
          auth:
            type: bearer
            token: "${API_TOKEN}"
      resources:
        memory_limit: 256Mi     # Memory limit
        timeout: 30s            # Request timeout
      scaling:
        min_instances: 0        # Minimum instances
        max_instances: 3        # Maximum instances
      depends_on: []            # Dependencies on other agents
      environment:              # Environment variables
        LOG_LEVEL: info
```

## API Reference

### Cluster Management

```bash
# List clusters
GET /api/v1/clusters

# Create cluster
POST /api/v1/clusters
Content-Type: application/json
{...cluster configuration...}

# Get cluster details
GET /api/v1/clusters/{name}

# Delete cluster
DELETE /api/v1/clusters/{name}

# Scale cluster agent
POST /api/v1/clusters/{name}/scale
{
  "agent": "agent-name",
  "instances": 3
}
```

### Agent Management

```bash
# List agents
GET /api/v1/agents

# Get agent details
GET /api/v1/agents/{id}

# Chat with agent
POST /api/v1/agents/{id}/chat
{
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}

# Stream chat with agent
POST /api/v1/agents/{id}/stream
```

### System Endpoints

```bash
# Health check
GET /health

# Readiness check
GET /ready

# System information
GET /api/v1/info

# Metrics
GET /api/v1/metrics

# Prometheus metrics
GET /metrics
```

## Tool Integration

### HTTP Tools

```yaml
tools:
  - type: http
    name: api_service
    url: "https://api.example.com"
    auth:
      type: bearer  # bearer, api_key, basic
      token: "${API_TOKEN}"
    config:
      header_User-Agent: "GoAgents/1.0"
```

### WebSocket Tools

```yaml
tools:
  - type: websocket
    name: live_chat
    endpoint: "wss://chat.example.com/ws"
    auth:
      type: bearer
      token: "${WS_TOKEN}"
    config:
      subprotocol: "chat-v1"
```

### MCP (Model Context Protocol) Tools

```yaml
tools:
  - type: mcp
    name: file_operations
    server: "file-mcp-server"
    config:
      timeout: "30s"
```

## Deployment

### Docker

```bash
# Build Docker image
docker build -t goagents:latest .

# Run with Docker
docker run -p 8080:8080 \
  -e ANTHROPIC_API_KEY=your-key \
  -v $(pwd)/config.yaml:/config.yaml \
  goagents:latest run --config /config.yaml
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goagents
spec:
  replicas: 1
  selector:
    matchLabels:
      app: goagents
  template:
    metadata:
      labels:
        app: goagents
    spec:
      containers:
      - name: goagents
        image: goagents:latest
        ports:
        - containerPort: 8080
        env:
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: llm-credentials
              key: anthropic-key
        volumeMounts:
        - name: config
          mountPath: /config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: goagents-config
```

### Cloud Run

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: goagents
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "0"
        autoscaling.knative.dev/maxScale: "10"
    spec:
      containers:
      - image: gcr.io/PROJECT/goagents:latest
        ports:
        - containerPort: 8080
        env:
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: llm-credentials
              key: anthropic-key
        resources:
          limits:
            memory: 512Mi
            cpu: 1000m
```

## Monitoring and Observability

### Metrics

GoAgents exposes Prometheus metrics at `/metrics`:

- `goagents_clusters_total`: Total number of clusters
- `goagents_agents_total`: Total number of agents
- `goagents_requests_total`: Total number of requests processed
- `goagents_request_duration_seconds`: Request duration histogram
- `goagents_agent_status`: Agent status by cluster and name

### Logging

Structured logging with configurable levels:

```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:00Z",
  "logger": "agent",
  "message": "Request processed",
  "cluster": "customer-support",
  "agent": "intent-classifier",
  "request_id": "req-123",
  "duration": "245ms"
}
```

### Health Checks

- `/health`: Basic health check
- `/ready`: Readiness check (includes cluster status)

## Examples

See the `examples/` directory for complete configuration examples:

- `customer-support.yaml`: Customer support chatbot cluster
- `data-analysis.json`: Data analysis pipeline cluster
- `config.yaml`: Basic server configuration

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- Documentation: [docs/](docs/)
- Issues: [GitHub Issues](https://github.com/goagents/goagents/issues)
- Discussions: [GitHub Discussions](https://github.com/goagents/goagents/discussions)