# API Reference

GoAgents provides a comprehensive REST API for managing agent clusters, agents, and interactions. All endpoints return JSON responses and follow RESTful conventions.

## Base URL

When running locally: `http://localhost:8080`

## Authentication

Currently, GoAgents operates without authentication for simplicity. In production environments, consider implementing API keys, JWT tokens, or OAuth2 depending on your security requirements.

## Response Format

All API responses follow this structure:

```json
{
  "success": true,
  "data": { ... },
  "message": "Optional message",
  "timestamp": "2025-01-30T16:15:08Z"
}
```

Error responses:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Detailed error message"
  },
  "timestamp": "2025-01-30T16:15:08Z"
}
```

## Health & Status Endpoints

### Health Check
Check if the service is running.

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-30T16:15:08Z"
}
```

### Readiness Check
Check if the service is ready to handle requests.

```http
GET /ready
```

**Response:**
```json
{
  "status": "ready",
  "timestamp": "2025-01-30T16:15:08Z"
}
```

## Cluster Management

### List Clusters
Get all deployed agent clusters.

```http
GET /api/v1/clusters
```

**Response:**
```json
{
  "success": true,
  "data": {
    "clusters": [
      {
        "name": "customer-support",
        "namespace": "default",
        "status": "running",
        "agents_count": 3,
        "created_at": "2025-01-30T16:15:08Z",
        "updated_at": "2025-01-30T16:15:08Z"
      }
    ]
  }
}
```

### Create Cluster
Deploy a new agent cluster.

```http
POST /api/v1/clusters
Content-Type: application/json
```

**Request Body:**
```json
{
  "apiVersion": "goagents.dev/v1",
  "kind": "AgentCluster",
  "metadata": {
    "name": "my-cluster",
    "namespace": "default"
  },
  "spec": {
    "resource_policy": {
      "max_concurrent_agents": 5,
      "idle_timeout": "300s",
      "scale_to_zero": true
    },
    "agents": [
      {
        "name": "assistant",
        "provider": "anthropic",
        "model": "claude-sonnet-4",
        "system_prompt": "You are a helpful assistant.",
        "tools": []
      }
    ]
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "cluster_id": "cluster-123",
    "name": "my-cluster",
    "status": "deploying"
  },
  "message": "Cluster deployment initiated"
}
```

### Get Cluster Details
Get detailed information about a specific cluster.

```http
GET /api/v1/clusters/{cluster_name}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "name": "customer-support",
    "namespace": "default",
    "status": "running",
    "agents": [
      {
        "name": "intent-classifier",
        "status": "running",
        "instances": 2,
        "provider": "anthropic",
        "model": "claude-sonnet-4"
      }
    ],
    "metrics": {
      "total_requests": 1250,
      "active_conversations": 5,
      "avg_response_time_ms": 150
    },
    "created_at": "2025-01-30T16:15:08Z",
    "updated_at": "2025-01-30T16:15:08Z"
  }
}
```

### Delete Cluster
Remove an agent cluster.

```http
DELETE /api/v1/clusters/{cluster_name}
```

**Response:**
```json
{
  "success": true,
  "message": "Cluster deleted successfully"
}
```

### Scale Cluster
Adjust the number of agent instances in a cluster.

```http
POST /api/v1/clusters/{cluster_name}/scale
Content-Type: application/json
```

**Request Body:**
```json
{
  "agent_name": "intent-classifier",
  "instances": 5
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "agent_name": "intent-classifier",
    "current_instances": 5,
    "target_instances": 5
  },
  "message": "Scaling operation completed"
}
```

## Agent Management

### List Agents
Get all active agents across clusters.

```http
GET /api/v1/agents
```

**Query Parameters:**
- `cluster` (optional): Filter by cluster name
- `status` (optional): Filter by status (running, idle, scaling)

**Response:**
```json
{
  "success": true,
  "data": {
    "agents": [
      {
        "id": "agent-123",
        "name": "intent-classifier",
        "cluster": "customer-support",
        "status": "running",
        "provider": "anthropic",
        "model": "claude-sonnet-4",
        "instances": 2,
        "active_conversations": 3,
        "created_at": "2025-01-30T16:15:08Z"
      }
    ]
  }
}
```

### Get Agent Details
Get detailed information about a specific agent.

```http
GET /api/v1/agents/{agent_id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "agent-123",
    "name": "intent-classifier",
    "cluster": "customer-support",
    "status": "running",
    "provider": "anthropic",
    "model": "claude-sonnet-4",
    "system_prompt": "You are an expert customer intent classifier...",
    "tools": [
      {
        "type": "http",
        "name": "customer_db",
        "url": "https://api.example.com/customers"
      }
    ],
    "metrics": {
      "total_requests": 450,
      "avg_response_time_ms": 120,
      "success_rate": 0.98
    },
    "instances": 2,
    "created_at": "2025-01-30T16:15:08Z"
  }
}
```

## Agent Interaction

### Chat with Agent
Send a message to an agent and receive a response.

```http
POST /api/v1/agents/{agent_id}/chat
Content-Type: application/json
```

**Request Body:**
```json
{
  "message": "Hello, I need help with my billing account.",
  "conversation_id": "conv-456",
  "context": {
    "user_id": "user-789",
    "session_id": "session-101"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "response": "Hello! I'd be happy to help you with your billing account. Could you please provide more details about the specific issue you're experiencing?",
    "conversation_id": "conv-456",
    "agent_id": "agent-123",
    "tokens_used": 45,
    "response_time_ms": 150,
    "timestamp": "2025-01-30T16:15:08Z"
  }
}
```

### Stream Chat with Agent
Stream a conversation with an agent using Server-Sent Events.

```http
POST /api/v1/agents/{agent_id}/stream
Content-Type: application/json
Accept: text/event-stream
```

**Request Body:**
```json
{
  "message": "Can you help me analyze this data?",
  "conversation_id": "conv-789"
}
```

**Response Stream:**
```
data: {"type": "start", "conversation_id": "conv-789"}

data: {"type": "token", "content": "I'd"}

data: {"type": "token", "content": " be"}

data: {"type": "token", "content": " happy"}

data: {"type": "token", "content": " to"}

data: {"type": "token", "content": " help"}

data: {"type": "end", "response": "I'd be happy to help you analyze the data. Please share the dataset or describe what you'd like to analyze.", "tokens_used": 25}
```

## Metrics & Monitoring

### System Metrics
Get system-wide metrics.

```http
GET /api/v1/metrics
```

**Response:**
```json
{
  "success": true,
  "data": {
    "system": {
      "uptime_seconds": 86400,
      "memory_usage_mb": 250,
      "cpu_usage_percent": 15,
      "goroutines": 45
    },
    "clusters": {
      "total": 3,
      "running": 3,
      "scaling": 0
    },
    "agents": {
      "total": 8,
      "active": 6,
      "idle": 2
    },
    "requests": {
      "total": 15420,
      "success_rate": 0.987,
      "avg_response_time_ms": 145,
      "requests_per_minute": 25
    }
  }
}
```

### Prometheus Metrics
Prometheus-compatible metrics endpoint.

```http
GET /metrics
```

**Response:**
```
# HELP goagents_requests_total Total number of requests processed
# TYPE goagents_requests_total counter
goagents_requests_total{agent="intent-classifier",status="success"} 1420

# HELP goagents_response_time_seconds Response time in seconds
# TYPE goagents_response_time_seconds histogram
goagents_response_time_seconds_bucket{agent="intent-classifier",le="0.1"} 450
goagents_response_time_seconds_bucket{agent="intent-classifier",le="0.5"} 1200
goagents_response_time_seconds_bucket{agent="intent-classifier",le="1.0"} 1400
goagents_response_time_seconds_bucket{agent="intent-classifier",le="+Inf"} 1420

# HELP goagents_active_agents Number of currently active agents
# TYPE goagents_active_agents gauge
goagents_active_agents{cluster="customer-support"} 3
```

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Request body is malformed or missing required fields |
| `CLUSTER_NOT_FOUND` | 404 | Specified cluster does not exist |
| `AGENT_NOT_FOUND` | 404 | Specified agent does not exist |
| `CLUSTER_EXISTS` | 409 | Cluster with the same name already exists |
| `SCALING_IN_PROGRESS` | 409 | Cannot modify cluster while scaling operation is active |
| `PROVIDER_ERROR` | 502 | Error communicating with AI provider |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

## Rate Limiting

Currently, GoAgents does not implement rate limiting. In production environments, consider implementing:

- Request rate limits per IP/API key
- Concurrent conversation limits per agent
- Token usage limits per time window

## WebSocket API (Future)

WebSocket support for real-time bidirectional communication is planned for a future release:

```javascript
// Planned WebSocket API
const ws = new WebSocket('ws://localhost:8080/api/v1/agents/agent-123/ws');
ws.send(JSON.stringify({
  type: 'message',
  content: 'Hello!',
  conversation_id: 'conv-123'
}));
```

## Examples

### Complete Chat Flow

```bash
# 1. Deploy a cluster
curl -X POST http://localhost:8080/api/v1/clusters \
  -H "Content-Type: application/json" \
  -d @my-cluster.json

# 2. List agents
curl http://localhost:8080/api/v1/agents

# 3. Start a conversation
curl -X POST http://localhost:8080/api/v1/agents/assistant/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello! How can you help me?",
    "conversation_id": "conv-001"
  }'

# 4. Continue the conversation
curl -X POST http://localhost:8080/api/v1/agents/assistant/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "I need help with data analysis",
    "conversation_id": "conv-001"
  }'
```

### JavaScript/Node.js Example

```javascript
const axios = require('axios');

class GoAgentsClient {
  constructor(baseURL = 'http://localhost:8080') {
    this.client = axios.create({ baseURL });
  }

  async deployCluster(clusterConfig) {
    const response = await this.client.post('/api/v1/clusters', clusterConfig);
    return response.data;
  }

  async chatWithAgent(agentId, message, conversationId) {
    const response = await this.client.post(`/api/v1/agents/${agentId}/chat`, {
      message,
      conversation_id: conversationId
    });
    return response.data;
  }

  async getMetrics() {
    const response = await this.client.get('/api/v1/metrics');
    return response.data;
  }
}

// Usage
const client = new GoAgentsClient();

async function example() {
  // Chat with an agent
  const response = await client.chatWithAgent(
    'assistant',
    'Hello, can you help me?',
    'conv-123'
  );
  
  console.log('Agent response:', response.data.response);
}
```

### Python Example

```python
import requests
import json

class GoAgentsClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
    
    def chat_with_agent(self, agent_id, message, conversation_id=None):
        url = f"{self.base_url}/api/v1/agents/{agent_id}/chat"
        payload = {
            "message": message,
            "conversation_id": conversation_id
        }
        response = requests.post(url, json=payload)
        return response.json()
    
    def get_clusters(self):
        url = f"{self.base_url}/api/v1/clusters"
        response = requests.get(url)
        return response.json()

# Usage
client = GoAgentsClient()

# Start a conversation
response = client.chat_with_agent(
    "assistant", 
    "Hello! Can you explain what GoAgents does?",
    "conv-001"
)

print("Agent response:", response["data"]["response"])
```

---

**Next**: [Configuration Reference](./configuration.md)