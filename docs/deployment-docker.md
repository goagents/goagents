# Docker Deployment Guide

This guide covers deploying GoAgents using Docker containers, from single-instance setups to production-ready configurations with monitoring and high availability.

## Quick Start with Docker

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- Environment variables set for your LLM providers

### Single Container Deployment

```bash
# Pull the latest image (or build locally)
docker pull goagents/goagents:latest

# Run with basic configuration
docker run -d \
  --name goagents \
  -p 8080:8080 \
  -p 9090:9090 \
  -e ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
  -e OPENAI_API_KEY="${OPENAI_API_KEY}" \
  -v $(pwd)/examples:/config \
  goagents/goagents:latest \
  run --config /config/config.yaml --cluster /config/customer-support.yaml
```

### Building from Source

```bash
# Build the Docker image
make docker-build

# Or manually
docker build -t goagents:local .

# Run your local build
docker run -d \
  --name goagents-local \
  -p 8080:8080 \
  -e ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY}" \
  -v $(pwd)/examples:/config \
  goagents:local
```

## Docker Compose Setup

### Basic Compose Configuration

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  goagents:
    image: goagents/goagents:latest
    ports:
      - "8080:8080"    # API server
      - "9090:9090"    # Metrics
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - GOOGLE_API_KEY=${GOOGLE_API_KEY}
      - GOAGENTS_LOG_LEVEL=info
    volumes:
      - ./config:/config:ro
      - ./logs:/var/log/goagents
    command: ["run", "--config", "/config/config.yaml", "--cluster", "/config/cluster.yaml"]
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

Deploy with:

```bash
# Start the services
docker-compose up -d

# View logs
docker-compose logs -f goagents

# Stop services
docker-compose down
```

### Production Compose with Monitoring

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  # GoAgents main service
  goagents:
    image: goagents/goagents:latest
    ports:
      - "8080:8080"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - GOOGLE_API_KEY=${GOOGLE_API_KEY}
      - GOAGENTS_LOG_LEVEL=warn
    volumes:
      - ./config:/config:ro
      - ./logs:/var/log/goagents
      - /etc/ssl/certs:/etc/ssl/certs:ro  # SSL certificates
    command: ["run", "--config", "/config/config-prod.yaml", "--cluster", "/config/cluster-prod.yaml"]
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '1.0'
        reservations:
          memory: 512M
          cpus: '0.5'
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    depends_on:
      - prometheus
      - grafana
    networks:
      - goagents-network

  # Prometheus for metrics collection
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=15d'
      - '--web.enable-lifecycle'
    restart: unless-stopped
    networks:
      - goagents-network

  # Grafana for visualization
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD:-admin}
      - GF_INSTALL_PLUGINS=grafana-piechart-panel
    volumes:
      - grafana-data:/var/lib/grafana
      - ./monitoring/grafana-datasources.yml:/etc/grafana/provisioning/datasources/datasources.yml:ro
      - ./monitoring/dashboards:/etc/grafana/provisioning/dashboards:ro
    restart: unless-stopped
    depends_on:
      - prometheus
    networks:
      - goagents-network

  # Redis for caching (optional)
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
    restart: unless-stopped
    networks:
      - goagents-network

  # Nginx reverse proxy
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - ./nginx/logs:/var/log/nginx
    depends_on:
      - goagents
    restart: unless-stopped
    networks:
      - goagents-network

volumes:
  prometheus-data:
  grafana-data:
  redis-data:

networks:
  goagents-network:
    driver: bridge
```

### Supporting Configuration Files

**Prometheus configuration** (`monitoring/prometheus.yml`):

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'goagents'
    static_configs:
      - targets: ['goagents:9090']
    scrape_interval: 10s
    metrics_path: /metrics

  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
```

**Nginx configuration** (`nginx/nginx.conf`):

```nginx
events {
    worker_connections 1024;
}

http {
    upstream goagents {
        server goagents:8080;
    }

    server {
        listen 80;
        server_name your-domain.com;
        
        # Redirect HTTP to HTTPS
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;

        # API endpoints
        location /api/ {
            proxy_pass http://goagents;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            
            # WebSocket support
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }

        # Health checks
        location /health {
            proxy_pass http://goagents;
        }

        # Metrics (restrict access)
        location /metrics {
            allow 10.0.0.0/8;
            allow 172.16.0.0/12;
            allow 192.168.0.0/16;
            deny all;
            proxy_pass http://goagents;
        }
    }
}
```

## Production Deployment Strategies

### Multi-Container Setup

For high availability, run multiple GoAgents instances:

```yaml
version: '3.8'

services:
  goagents-1:
    image: goagents/goagents:latest
    ports:
      - "8081:8080"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GOAGENTS_NODE_ID=node-1
    volumes:
      - ./config:/config:ro
    networks:
      - goagents-network

  goagents-2:
    image: goagents/goagents:latest
    ports:
      - "8082:8080"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GOAGENTS_NODE_ID=node-2
    volumes:
      - ./config:/config:ro
    networks:
      - goagents-network

  goagents-3:
    image: goagents/goagents:latest
    ports:
      - "8083:8080"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GOAGENTS_NODE_ID=node-3
    volumes:
      - ./config:/config:ro
    networks:
      - goagents-network

  # Load balancer
  haproxy:
    image: haproxy:alpine
    ports:
      - "8080:8080"
      - "8404:8404"  # Stats page
    volumes:
      - ./haproxy/haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro
    depends_on:
      - goagents-1
      - goagents-2
      - goagents-3
    networks:
      - goagents-network

networks:
  goagents-network:
    driver: bridge
```

**HAProxy configuration** (`haproxy/haproxy.cfg`):

```
global
    daemon
    maxconn 4096

defaults
    mode http
    timeout connect 5000ms
    timeout client 50000ms
    timeout server 50000ms

frontend goagents_frontend
    bind *:8080
    default_backend goagents_backend

backend goagents_backend
    balance roundrobin
    option httpchk GET /health
    server goagents-1 goagents-1:8080 check
    server goagents-2 goagents-2:8080 check
    server goagents-3 goagents-3:8080 check

frontend stats
    bind *:8404
    stats enable
    stats uri /stats
    stats refresh 30s
```

### Docker Swarm Deployment

For container orchestration with Docker Swarm:

```yaml
# docker-stack.yml
version: '3.8'

services:
  goagents:
    image: goagents/goagents:latest
    ports:
      - target: 8080
        published: 8080
        protocol: tcp
        mode: ingress
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    volumes:
      - type: bind
        source: ./config
        target: /config
        read_only: true
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
        order: start-first
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      resources:
        limits:
          memory: 1G
          cpus: '1.0'
        reservations:
          memory: 512M
          cpus: '0.5'
      placement:
        constraints:
          - node.role == worker
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - goagents-overlay

  visualizer:
    image: dockersamples/visualizer:stable
    ports:
      - "8080:8080"
    volumes:
      - "/var/run/docker.sock:/var/run/docker.sock"
    deploy:
      placement:
        constraints: [node.role == manager]
    networks:
      - goagents-overlay

networks:
  goagents-overlay:
    driver: overlay
    attachable: true
```

Deploy to swarm:

```bash
# Initialize swarm (if not already done)
docker swarm init

# Deploy the stack
docker stack deploy -c docker-stack.yml goagents

# Check services
docker service ls

# Scale services
docker service scale goagents_goagents=5

# Remove stack
docker stack rm goagents
```

## Security Considerations

### Secure Docker Configuration

**Use non-root user**:

```dockerfile
# In your Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o goagents cmd/goagents/main.go

FROM alpine:latest
RUN addgroup -g 1001 -S goagents && \
    adduser -u 1001 -S goagents -G goagents
USER goagents
WORKDIR /home/goagents
COPY --from=builder /app/goagents .
EXPOSE 8080
CMD ["./goagents"]
```

**Secrets management**:

```yaml
# Using Docker secrets
version: '3.8'

services:
  goagents:
    image: goagents/goagents:latest
    secrets:
      - anthropic_api_key
      - openai_api_key
    environment:
      - ANTHROPIC_API_KEY_FILE=/run/secrets/anthropic_api_key
      - OPENAI_API_KEY_FILE=/run/secrets/openai_api_key

secrets:
  anthropic_api_key:
    file: ./secrets/anthropic_api_key.txt
  openai_api_key:
    file: ./secrets/openai_api_key.txt
```

### Network Security

```yaml
# Isolated networks
version: '3.8'

services:
  goagents:
    networks:
      - frontend
      - backend
  
  database:
    networks:
      - backend

  nginx:
    networks:
      - frontend

networks:
  frontend:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
  backend:
    driver: bridge
    internal: true
```

## Monitoring and Logging

### Centralized Logging

```yaml
version: '3.8'

services:
  goagents:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
        
  # ELK Stack for log aggregation
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:7.14.0
    environment:
      - discovery.type=single-node
    volumes:
      - elasticsearch-data:/usr/share/elasticsearch/data

  logstash:
    image: docker.elastic.co/logstash/logstash:7.14.0
    volumes:
      - ./logstash/pipeline:/usr/share/logstash/pipeline:ro

  kibana:
    image: docker.elastic.co/kibana/kibana:7.14.0
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch

volumes:
  elasticsearch-data:
```

### Health Checks and Monitoring

```bash
# Health check script
#!/bin/bash
# health-check.sh

ENDPOINT="http://localhost:8080/health"
TIMEOUT=5

if curl -f -s --max-time $TIMEOUT $ENDPOINT > /dev/null; then
    echo "GoAgents is healthy"
    exit 0
else
    echo "GoAgents health check failed"
    exit 1
fi
```

```yaml
# In docker-compose.yml
healthcheck:
  test: ["CMD", "/health-check.sh"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

## Troubleshooting

### Common Docker Issues

**Container won't start**:
```bash
# Check container logs
docker logs goagents

# Check container status
docker ps -a

# Inspect container
docker inspect goagents
```

**Permission issues**:
```bash
# Fix volume permissions
sudo chown -R 1001:1001 ./config ./logs

# Or use bind mounts with correct user
docker run --user $(id -u):$(id -g) ...
```

**Network connectivity**:
```bash
# Test network connectivity
docker exec goagents curl -f http://localhost:8080/health

# Check network configuration
docker network ls
docker network inspect goagents_default
```

### Performance Tuning

**Memory optimization**:
```yaml
services:
  goagents:
    deploy:
      resources:
        limits:
          memory: 2G
        reservations:
          memory: 1G
    environment:
      - GOGC=100  # Go garbage collection
      - GOMAXPROCS=4  # Max Go processes
```

**Container optimization**:
```dockerfile
# Multi-stage build for smaller images
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o goagents cmd/goagents/main.go

FROM scratch
COPY --from=builder /app/goagents /goagents
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
CMD ["/goagents"]
```

## Backup and Recovery

### Configuration Backup

```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

# Backup configurations
cp -r ./config $BACKUP_DIR/
cp -r ./monitoring $BACKUP_DIR/
cp docker-compose.yml $BACKUP_DIR/

# Backup volumes
docker run --rm -v goagents_prometheus-data:/data -v $BACKUP_DIR:/backup alpine tar czf /backup/prometheus-data.tar.gz -C /data .
docker run --rm -v goagents_grafana-data:/data -v $BACKUP_DIR:/backup alpine tar czf /backup/grafana-data.tar.gz -C /data .

echo "Backup completed: $BACKUP_DIR"
```

### Disaster Recovery

```bash
#!/bin/bash
# restore.sh

BACKUP_DIR=$1

if [ -z "$BACKUP_DIR" ]; then
    echo "Usage: $0 <backup_directory>"
    exit 1
fi

# Stop services
docker-compose down

# Restore configurations
cp -r $BACKUP_DIR/config ./
cp -r $BACKUP_DIR/monitoring ./
cp $BACKUP_DIR/docker-compose.yml ./

# Restore volumes
docker run --rm -v goagents_prometheus-data:/data -v $BACKUP_DIR:/backup alpine tar xzf /backup/prometheus-data.tar.gz -C /data
docker run --rm -v goagents_grafana-data:/data -v $BACKUP_DIR:/backup alpine tar xzf /backup/grafana-data.tar.gz -C /data

# Start services
docker-compose up -d

echo "Restore completed from: $BACKUP_DIR"
```

---

**Next**: [Kubernetes Deployment](./deployment-kubernetes.md)