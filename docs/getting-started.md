# Getting Started with GoAgents

This guide will help you get up and running with GoAgents in just a few minutes.

## Prerequisites

- Go 1.21 or later
- One or more LLM provider API keys:
  - Anthropic Claude API key
  - OpenAI API key
  - Google AI Studio API key

## Quick Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/goagents/goagents.git
cd goagents

# Build the binary
make build

# The binary is now available as ./goagents
./goagents --help
```

### Option 2: Using Go Install

```bash
# Install directly from Go modules
go install ./cmd/goagents

# Now available globally
goagents --help
```

## Environment Setup

Set up your API keys as environment variables:

```bash
# Required: At least one provider API key
export ANTHROPIC_API_KEY="your-anthropic-api-key"
export OPENAI_API_KEY="your-openai-api-key"
export GOOGLE_API_KEY="your-google-api-key"

# Optional
export OPENAI_ORG_ID="your-openai-org-id"
export GOOGLE_PROJECT_ID="your-google-project-id"
```

## Your First Agent Cluster

### 1. Create a Basic Configuration

Create `my-config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  timeout: 30s
  log_level: info

providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"
    version: "2023-06-01"
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com"
    org_id: "${OPENAI_ORG_ID}"
```

### 2. Create an Agent Cluster

Create `my-cluster.yaml`:

```yaml
apiVersion: goagents.dev/v1
kind: AgentCluster
metadata:
  name: hello-world
  namespace: default
spec:
  resource_policy:
    max_concurrent_agents: 3
    idle_timeout: 300s
    scale_to_zero: true
  agents:
    - name: assistant
      provider: anthropic
      model: claude-sonnet-4
      system_prompt: |
        You are a helpful AI assistant. Be concise and friendly.
        Answer questions clearly and provide helpful information.
      tools: []
```

### 3. Start GoAgents

```bash
# Start the GoAgents server
./goagents run --config my-config.yaml --cluster my-cluster.yaml
```

You should see output like:

```
2025-01-30T16:15:08Z INFO Starting GoAgents server {"version": "1.0.0", "port": 8080}
2025-01-30T16:15:08Z INFO Agent cluster deployed successfully {"cluster": "hello-world", "agents": 1}
2025-01-30T16:15:08Z INFO Server listening {"address": "0.0.0.0:8080"}
```

### 4. Test Your Setup

In another terminal, test the health endpoint:

```bash
curl http://localhost:8080/health
# Should return: {"status":"healthy","timestamp":"2025-01-30T16:15:08Z"}
```

List your agents:

```bash
curl http://localhost:8080/api/v1/agents
```

Chat with your agent:

```bash
curl -X POST http://localhost:8080/api/v1/agents/assistant/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello! Can you tell me about GoAgents?"}'
```

## Using the CLI

GoAgents provides a comprehensive CLI for managing your agent clusters:

```bash
# Deploy a cluster
goagents deploy --cluster my-cluster.yaml

# Check cluster status
goagents status --cluster hello-world

# Scale agents
goagents scale --cluster hello-world --agent assistant --instances 2

# View logs
goagents logs --cluster hello-world --follow

# Delete cluster
goagents delete --cluster hello-world
```

## Next Steps

Now that you have GoAgents running, explore these topics:

1. **[Configuration](./configuration.md)** - Learn about advanced configuration options
2. **[Tools](./tools.md)** - Connect your agents to external tools and APIs
3. **[API Reference](./api-reference.md)** - Explore the full REST API
4. **[Deployment](./deployment-docker.md)** - Deploy to production environments

## Example Use Cases

### Customer Support Bot

```yaml
apiVersion: goagents.dev/v1
kind: AgentCluster
metadata:
  name: customer-support
spec:
  agents:
    - name: intent-classifier
      provider: anthropic
      model: claude-sonnet-4
      system_prompt: |
        You are a customer support intent classifier. 
        Classify customer requests into: billing, technical, general, escalation.
      
    - name: billing-agent
      provider: openai
      model: gpt-4
      system_prompt: |
        You are a billing support specialist. Help customers with:
        - Account inquiries
        - Payment issues
        - Subscription changes
      tools:
        - type: http
          name: billing_api
          url: "https://api.company.com/billing"
```

### Data Analysis Pipeline

```yaml
apiVersion: goagents.dev/v1
kind: AgentCluster
metadata:
  name: data-analysis
spec:
  agents:
    - name: data-processor
      provider: gemini
      model: gemini-pro
      system_prompt: |
        You are a data analysis expert. Process datasets and provide insights.
      tools:
        - type: mcp
          name: python_executor
          server: "python-mcp-server"
        
    - name: report-generator
      provider: anthropic
      model: claude-sonnet-4
      system_prompt: |
        You generate clear, executive-level reports from data analysis results.
```

## Troubleshooting

### Common Issues

**"No API key found"**
- Ensure your environment variables are set correctly
- Check that the variable names match exactly (case-sensitive)

**"Connection refused"**
- Verify the server is running on the expected port
- Check firewall settings if running on remote host

**"Agent not responding"**
- Check the logs: `goagents logs --cluster your-cluster --follow`
- Verify your LLM provider API keys are valid
- Check provider service status

### Getting Help

- Check the [Troubleshooting Guide](./troubleshooting.md)
- Review logs for error messages
- File an issue on [GitHub](https://github.com/goagents/goagents/issues)

---

**Next**: [Configuration Reference](./configuration.md)