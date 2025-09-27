# Deployment Guide

## Production Deployment

### Environment Setup

1. **Server Requirements:**
   - Ubuntu 20.04 LTS or higher
   - 2+ CPU cores
   - 4GB+ RAM
   - 20GB+ storage

2. **Install Dependencies:**
\`\`\`bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install PostgreSQL
sudo apt install postgresql postgresql-contrib -y
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Install Git
sudo apt install git -y
\`\`\`

3. **Database Setup:**
\`\`\`bash
sudo -u postgres psql
CREATE DATABASE assignment;
CREATE USER stocky_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE assignment TO stocky_user;
\q
\`\`\`

4. **Application Deployment:**
\`\`\`bash
# Clone repository
git clone <repository-url>
cd stocky-backend

# Build application
go build -o stocky main.go

# Create systemd service
sudo tee /etc/systemd/system/stocky.service > /dev/null <<EOF
[Unit]
Description=Stocky Stock Rewards API
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/stocky-backend
ExecStart=/home/ubuntu/stocky-backend/stocky
Restart=always
RestartSec=5
Environment=DATABASE_URL=postgres://stocky_user:secure_password@localhost/assignment?sslmode=disable
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable stocky
sudo systemctl start stocky
\`\`\`

### Docker Deployment

1. **Create Dockerfile:**
\`\`\`dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o stocky main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/stocky .
COPY --from=builder /app/scripts ./scripts

CMD ["./stocky"]
\`\`\`

2. **Create docker-compose.yml:**
\`\`\`yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: assignment
      POSTGRES_USER: stocky_user
      POSTGRES_PASSWORD: secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  stocky:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://stocky_user:secure_password@postgres/assignment?sslmode=disable
      PORT: 8080
    depends_on:
      - postgres

volumes:
  postgres_data:
\`\`\`

3. **Deploy with Docker:**
\`\`\`bash
docker-compose up -d
\`\`\`

### Kubernetes Deployment

1. **Create ConfigMap:**
\`\`\`yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: stocky-config
data:
  DATABASE_URL: "postgres://stocky_user:secure_password@postgres-service/assignment?sslmode=disable"
  PORT: "8080"
\`\`\`

2. **Create Deployment:**
\`\`\`yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stocky-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: stocky
  template:
    metadata:
      labels:
        app: stocky
    spec:
      containers:
      - name: stocky
        image: stocky:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: stocky-config
\`\`\`

3. **Create Service:**
\`\`\`yaml
apiVersion: v1
kind: Service
metadata:
  name: stocky-service
spec:
  selector:
    app: stocky
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
\`\`\`

## Monitoring and Logging

### Prometheus Metrics

Add metrics collection:
\`\`\`go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// Add to main.go
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
\`\`\`

### Log Aggregation

Configure structured logging for production:
\`\`\`go
logrus.SetFormatter(&logrus.JSONFormatter{})
logrus.SetOutput(os.Stdout)
\`\`\`

### Health Checks

The `/health` endpoint provides basic health checking. For production, consider adding:
- Database connectivity check
- External service dependency checks
- Resource utilization metrics

## Security Hardening

1. **Environment Variables:**
   - Never commit secrets to version control
   - Use environment-specific configuration
   - Implement secret rotation

2. **Database Security:**
   - Use connection pooling
   - Enable SSL/TLS for database connections
   - Regular security updates

3. **API Security:**
   - Implement rate limiting
   - Add authentication/authorization
   - Input validation and sanitization
   - CORS configuration

4. **Network Security:**
   - Use HTTPS in production
   - Implement firewall rules
   - VPN for database access

## Backup and Recovery

1. **Database Backups:**
\`\`\`bash
# Daily backup script
#!/bin/bash
pg_dump -h localhost -U stocky_user assignment > backup_$(date +%Y%m%d).sql
\`\`\`

2. **Application Backups:**
   - Version control for code
   - Configuration backups
   - Log retention policies

## Performance Optimization

1. **Database Optimization:**
   - Regular VACUUM and ANALYZE
   - Index optimization
   - Connection pooling

2. **Application Optimization:**
   - Caching strategies
   - Async processing for heavy operations
   - Load balancing

3. **Monitoring:**
   - Response time monitoring
   - Error rate tracking
   - Resource utilization alerts
