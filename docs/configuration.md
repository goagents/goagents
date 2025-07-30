# Configuration Reference

GoAgents uses YAML configuration files to define server settings, provider configurations, and agent clusters. This guide covers all available configuration options.

## Configuration File Structure

GoAgents uses two main configuration files:

1. **Server Configuration** (`config.yaml`) - Server settings and provider credentials
2. **Cluster Configuration** (`cluster.yaml`) - Agent cluster definitions

## Server Configuration (`config.yaml`)

### Basic Structure

```yaml
# Server configuration
server:
  host: "0.0.0.0"
  port: 8080
  timeout: 30s
  log_level: info
  metrics:
    enabled: true
    path: /metrics
    port: 9090

# Provider configurations
providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"
    version: "2023-06-01"
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com"
    org_id: "${OPENAI_ORG_ID}"
  gemini:
    api_key: "${GOOGLE_API_KEY}"
    project_id: "${GOOGLE_PROJECT_ID}"

# Logging configuration
logging:
  level: info
  format: json
  output: stdout
  file: /var/log/goagents.log

# Security settings
security:
  enable_cors: true
  cors_origins: ["*"]
  rate_limiting:
    enabled: false
    requests_per_minute: 100
```

### Server Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `host` | string | `"0.0.0.0"` | Server bind address |
| `port` | int | `8080` | Server port |
| `timeout` | duration | `30s` | Request timeout |
| `log_level` | string | `info` | Log level (debug, info, warn, error) |
| `read_timeout` | duration | `30s` | HTTP read timeout |
| `write_timeout` | duration | `30s` | HTTP write timeout |
| `idle_timeout` | duration | `60s` | HTTP idle timeout |

### Metrics Section

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable Prometheus metrics |
| `path` | string | `"/metrics"` | Metrics endpoint path |
| `port` | int | `9090` | Metrics server port |

### Provider Configurations

#### Anthropic Provider

```yaml
providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"           # Required: API key
    base_url: "https://api.anthropic.com"     # Optional: Custom base URL
    version: "2023-06-01"                     # Optional: API version
    timeout: 30s                              # Optional: Request timeout
    max_retries: 3                            # Optional: Max retry attempts
    retry_delay: 1s                           # Optional: Delay between retries
```

#### OpenAI Provider

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"              # Required: API key
    base_url: "https://api.openai.com"        # Optional: Custom base URL
    org_id: "${OPENAI_ORG_ID}"               # Optional: Organization ID
    timeout: 30s                              # Optional: Request timeout
    max_retries: 3                            # Optional: Max retry attempts
    retry_delay: 1s                           # Optional: Delay between retries
```

#### Google Gemini Provider

```yaml
providers:
  gemini:
    api_key: "${GOOGLE_API_KEY}"              # Required: API key
    project_id: "${GOOGLE_PROJECT_ID}"        # Optional: Project ID
    base_url: "https://generativelanguage.googleapis.com" # Optional: Custom base URL
    timeout: 30s                              # Optional: Request timeout
    max_retries: 3                            # Optional: Max retry attempts
    retry_delay: 1s                           # Optional: Delay between retries
```

### Logging Configuration

```yaml
logging:
  level: info                    # Log level: debug, info, warn, error
  format: json                   # Format: json, text
  output: stdout                 # Output: stdout, stderr, file
  file: /var/log/goagents.log   # Log file path (when output=file)
  max_size: 100                  # Max log file size in MB
  max_backups: 5                 # Max number of backup files
  max_age: 30                    # Max age of log files in days
  compress: true                 # Compress backup files
```

### Security Configuration

```yaml
security:
  enable_cors: true                    # Enable CORS
  cors_origins: ["*"]                  # Allowed CORS origins
  cors_methods: ["GET", "POST", "PUT", "DELETE"] # Allowed methods
  cors_headers: ["*"]                  # Allowed headers
  
  # Rate limiting (future feature)
  rate_limiting:
    enabled: false                     # Enable rate limiting
    requests_per_minute: 100           # Requests per minute limit
    burst_size: 10                     # Burst size
    
  # Authentication (future feature)
  auth:
    enabled: false                     # Enable authentication
    method: "api_key"                  # Method: api_key, jwt, oauth2
    api_keys: []                       # Valid API keys
```

## Cluster Configuration

### Basic Structure

```yaml
apiVersion: goagents.dev/v1
kind: AgentCluster
metadata:
  name: my-cluster
  namespace: default
  labels:
    environment: production
    team: ai-team
  annotations:
    description: "Customer support agent cluster"
    
spec:
  # Resource management policy
  resource_policy:
    max_concurrent_agents: 10
    idle_timeout: 300s
    scale_to_zero: true
    memory_limit: "512Mi"
    cpu_limit: "500m"
    
  # Agent definitions
  agents:
    - name: intent-classifier
      provider: anthropic
      model: claude-sonnet-4
      system_prompt: |
        You are an expert customer intent classifier.
        Classify customer requests into categories.
      tools:
        - type: http
          name: customer_db
          url: "https://api.example.com/customers"
        - type: mcp
          name: knowledge_base
          server: "knowledge-base-mcp-server"
      scaling:
        min_instances: 1
        max_instances: 5
        target_utilization: 0.7
```

### Metadata Section

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique cluster name |
| `namespace` | string | Kubernetes-style namespace |
| `labels` | map | Key-value labels for organization |
| `annotations` | map | Additional metadata |

### Resource Policy

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_concurrent_agents` | int | `10` | Maximum concurrent agents |
| `idle_timeout` | duration | `300s` | Agent idle timeout |
| `scale_to_zero` | bool | `true` | Allow scaling to zero |
| `memory_limit` | string | `"512Mi"` | Memory limit per agent |
| `cpu_limit` | string | `"500m"` | CPU limit per agent |

### Agent Configuration

#### Basic Agent

```yaml
agents:
  - name: my-agent
    provider: anthropic              # Required: LLM provider
    model: claude-sonnet-4          # Required: Model name
    system_prompt: |                # Required: System prompt
      You are a helpful AI assistant.
    temperature: 0.7                # Optional: Response randomness (0.0-1.0)
    max_tokens: 1000               # Optional: Max response tokens
    top_p: 0.9                     # Optional: Top-p sampling
    tools: []                      # Optional: Connected tools
    scaling:                       # Optional: Scaling configuration
      min_instances: 1
      max_instances: 3
      target_utilization: 0.8
```

#### Agent Scaling Configuration

```yaml
scaling:
  min_instances: 1                 # Minimum instances
  max_instances: 10                # Maximum instances
  target_utilization: 0.8          # Target CPU/memory utilization
  scale_up_threshold: 5            # Requests to trigger scale up
  scale_down_threshold: 2          # Requests to trigger scale down
  cooldown_period: 60s             # Time between scaling operations
```

### Tool Configurations

#### HTTP Tool

```yaml
tools:
  - type: http
    name: api_service                # Tool name
    url: "https://api.example.com"   # Base URL
    method: POST                     # HTTP method
    headers:                         # Custom headers
      Authorization: "Bearer ${API_TOKEN}"
      Content-Type: "application/json"
    timeout: 10s                     # Request timeout
    retry_attempts: 3                # Retry attempts
    retry_delay: 1s                  # Retry delay
```

#### MCP (Model Context Protocol) Tool

```yaml
tools:
  - type: mcp
    name: python_executor            # Tool name
    server: "python-mcp-server"      # MCP server identifier
    transport: stdio                 # Transport: stdio, websocket, http
    command: ["python", "-m", "mcp_server"] # Server command
    args: ["--port", "8080"]         # Server arguments
    env:                             # Environment variables
      PYTHON_PATH: "/usr/bin/python"
    timeout: 30s                     # Connection timeout
```

#### WebSocket Tool

```yaml
tools:
  - type: websocket
    name: realtime_data             # Tool name
    url: "wss://api.example.com/ws" # WebSocket URL
    headers:                        # Connection headers
      Authorization: "Bearer ${WS_TOKEN}"
    reconnect: true                 # Auto-reconnect
    reconnect_interval: 5s          # Reconnect interval
    ping_interval: 30s              # Ping interval
    max_message_size: 1048576       # Max message size (bytes)
```

## Environment Variables

GoAgents supports environment variable substitution in configuration files using `${VARIABLE_NAME}` syntax.

### Standard Environment Variables

```bash
# Provider API Keys
export ANTHROPIC_API_KEY="your-anthropic-key"
export OPENAI_API_KEY="your-openai-key"
export GOOGLE_API_KEY="your-google-key"
export OPENAI_ORG_ID="your-openai-org-id"
export GOOGLE_PROJECT_ID="your-google-project-id"

# Server Configuration
export GOAGENTS_HOST="0.0.0.0"
export GOAGENTS_PORT="8080"
export GOAGENTS_LOG_LEVEL="info"

# Tool Authentication
export API_TOKEN="your-api-token"
export WS_TOKEN="your-websocket-token"
```

## Configuration Validation

GoAgents validates configurations at startup and provides detailed error messages:

```bash
# Validate configuration without starting
goagents validate --config config.yaml --cluster cluster.yaml

# Start with validation
goagents run --config config.yaml --cluster cluster.yaml --validate
```

Common validation errors:
- Missing required fields
- Invalid duration formats
- Unknown provider or model names
- Invalid tool configurations

## Hot Reload

GoAgents supports hot reloading of configurations without restart:

```bash
# Send SIGHUP to reload configuration
kill -HUP $(pgrep goagents)

# Or use the CLI
goagents reload --config config.yaml --cluster cluster.yaml
```

## Advanced Examples

### Multi-Environment Configuration

**Production (`config-prod.yaml`)**:
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  log_level: warn

providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY_PROD}"
    timeout: 60s
    max_retries: 5

logging:
  level: warn
  format: json
  output: file
  file: /var/log/goagents-prod.log

security:
  enable_cors: true
  cors_origins: ["https://myapp.com"]
```

**Development (`config-dev.yaml`)**:
```yaml
server:
  host: "localhost"
  port: 8080
  log_level: debug

providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY_DEV}"
    timeout: 30s

logging:
  level: debug
  format: text
  output: stdout
```

### Complex Agent Cluster

```yaml
apiVersion: goagents.dev/v1
kind: AgentCluster
metadata:
  name: enterprise-support
  namespace: production
  labels:
    tier: premium
    region: us-west-2
    
spec:
  resource_policy:
    max_concurrent_agents: 50
    idle_timeout: 600s
    scale_to_zero: false
    memory_limit: "1Gi"
    cpu_limit: "1000m"
    
  agents:
    # Tier 1 Support
    - name: l1-support
      provider: anthropic
      model: claude-sonnet-4
      system_prompt: |
        You are a Tier 1 customer support agent. Handle basic inquiries
        and escalate complex issues to appropriate specialists.
      tools:
        - type: http
          name: crm_system
          url: "https://crm.company.com/api"
          headers:
            Authorization: "Bearer ${CRM_TOKEN}"
        - type: mcp
          name: ticket_system
          server: "ticketing-mcp-server"
      scaling:
        min_instances: 3
        max_instances: 15
        target_utilization: 0.8
        
    # Technical Support
    - name: tech-support
      provider: openai
      model: gpt-4
      system_prompt: |
        You are a technical support specialist. Help customers with
        complex technical issues and product integrations.
      tools:
        - type: http
          name: product_api
          url: "https://api.company.com/v1"
        - type: websocket
          name: system_monitoring
          url: "wss://monitoring.company.com/ws"
      scaling:
        min_instances: 2
        max_instances: 8
        target_utilization: 0.7
        
    # Billing Support
    - name: billing-support
      provider: gemini
      model: gemini-pro
      system_prompt: |
        You are a billing support specialist. Handle account inquiries,
        payment issues, and subscription management.
      tools:
        - type: http
          name: billing_system
          url: "https://billing.company.com/api"
          headers:
            Authorization: "Bearer ${BILLING_TOKEN}"
      scaling:
        min_instances: 1
        max_instances: 5
        target_utilization: 0.6
```

### Tool-Heavy Configuration

```yaml
apiVersion: goagents.dev/v1
kind: AgentCluster
metadata:
  name: data-analysis-cluster
  
spec:
  agents:
    - name: data-analyst
      provider: anthropic
      model: claude-sonnet-4
      system_prompt: |
        You are a data analysis expert with access to multiple tools.
        Help users analyze data, create visualizations, and generate insights.
      tools:
        # Database access
        - type: http
          name: postgres_api
          url: "https://db-api.company.com"
          headers:
            Authorization: "Bearer ${DB_TOKEN}"
            
        # Python execution
        - type: mcp
          name: python_executor
          server: "python-mcp-server"
          command: ["python", "-m", "mcp_python_server"]
          env:
            PYTHONPATH: "/opt/analysis/lib"
            
        # Visualization service
        - type: websocket
          name: viz_service
          url: "wss://viz.company.com/ws"
          reconnect: true
          
        # File storage
        - type: http
          name: file_storage
          url: "https://storage.company.com/api"
          method: POST
          headers:
            Authorization: "Bearer ${STORAGE_TOKEN}"
```

## Configuration Best Practices

1. **Use Environment Variables**: Never hardcode secrets in configuration files
2. **Validate Early**: Always validate configurations before deployment
3. **Start Small**: Begin with simple configurations and gradually add complexity
4. **Monitor Resources**: Set appropriate resource limits based on your infrastructure
5. **Plan for Scale**: Consider scaling patterns when designing agent clusters
6. **Organize by Environment**: Use separate configuration files for different environments
7. **Document Custom Tools**: Clearly document any custom tool integrations

---

**Next**: [Deployment Guide](./deployment-docker.md)