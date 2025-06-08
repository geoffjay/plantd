[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/logger)](https://goreportcard.com/report/github.com/geoffjay/plantd/logger)

---

# 🪵 Logger Service

The logger service provides centralized logging, event tracking, and audit trails for the plantd distributed control system. It uses TimescaleDB as a time series database to efficiently store and query logs, metrics, and alerts across all system components.

## Features

- **Centralized Logging**: Collect logs from all plantd services in one location
- **Time Series Storage**: Efficient storage and querying using TimescaleDB
- **Structured Logging**: Support for JSON and structured log formats
- **Log Aggregation**: Aggregate logs by service, level, and time periods
- **Real-time Monitoring**: Live log streaming and real-time alerts
- **Audit Trails**: Comprehensive audit logging for security and compliance
- **Log Retention**: Configurable log retention policies and archiving
- **Search and Filtering**: Advanced log search and filtering capabilities

## Quick Start

### Prerequisites

- Go 1.24 or later
- PostgreSQL with TimescaleDB extension
- ZeroMQ library (for message bus integration)

### Installation

```bash
# Install TimescaleDB (Ubuntu/Debian)
sudo apt-get install timescaledb-postgresql-14

# Enable TimescaleDB extension
sudo -u postgres psql -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"

# Build the logger service
make build-logger
```

### Basic Usage

```bash
# Start with default configuration
./build/plantd-logger

# Start with debug logging
PLANTD_LOGGER_LOG_LEVEL=debug ./build/plantd-logger

# Start with custom database URL
PLANTD_LOGGER_DB_URL="postgres://user:pass@localhost/plantd_logs" ./build/plantd-logger
```

## Configuration

### Environment Variables

```bash
# Database configuration
export PLANTD_LOGGER_DB_URL="postgres://user:pass@localhost/plantd_logs"
export PLANTD_LOGGER_DB_MAX_CONNECTIONS="10"

# Message bus configuration
export PLANTD_LOGGER_BUS_ENDPOINT="tcp://localhost:11001"

# Service configuration
export PLANTD_LOGGER_PORT="8082"
export PLANTD_LOGGER_HOST="0.0.0.0"

# Log retention
export PLANTD_LOGGER_RETENTION_DAYS="30"
export PLANTD_LOGGER_ARCHIVE_ENABLED="true"

# Log level
export PLANTD_LOGGER_LOG_LEVEL="info"
```

### Configuration File

```yaml
# config/logger.yaml
database:
  url: "postgres://user:pass@localhost/plantd_logs"
  max_connections: 10
  ssl_mode: "require"

server:
  port: 8082
  host: "0.0.0.0"

bus:
  endpoint: "tcp://localhost:11001"
  topics:
    - "logs.*"
    - "events.*"
    - "metrics.*"

retention:
  days: 30
  archive_enabled: true
  archive_path: "/var/log/plantd/archive"

log:
  level: "info"
  format: "json"
```

## Database Schema

The logger service uses TimescaleDB hypertables for efficient time series storage:

```sql
-- Main logs table (hypertable)
CREATE TABLE logs (
    time TIMESTAMPTZ NOT NULL,
    service VARCHAR(255) NOT NULL,
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    context JSONB,
    host VARCHAR(255),
    pid INTEGER,
    correlation_id UUID
);

-- Convert to hypertable
SELECT create_hypertable('logs', 'time');

-- Events table for structured events
CREATE TABLE events (
    time TIMESTAMPTZ NOT NULL,
    service VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    severity VARCHAR(20) DEFAULT 'info',
    correlation_id UUID
);

SELECT create_hypertable('events', 'time');

-- Metrics table for numerical data
CREATE TABLE metrics (
    time TIMESTAMPTZ NOT NULL,
    service VARCHAR(255) NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    tags JSONB,
    unit VARCHAR(50)
);

SELECT create_hypertable('metrics', 'time');

-- Audit table for security events
CREATE TABLE audit_log (
    time TIMESTAMPTZ NOT NULL,
    user_id UUID,
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255),
    result VARCHAR(50) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    details JSONB
);

SELECT create_hypertable('audit_log', 'time');
```

## API Endpoints

### Log Ingestion

```bash
# Submit log entry
POST /api/v1/logs
{
  "service": "org.plantd.Broker",
  "level": "info",
  "message": "Service started successfully",
  "context": {
    "port": 7200,
    "endpoint": "tcp://*:7200"
  }
}

# Submit event
POST /api/v1/events
{
  "service": "org.plantd.State",
  "event_type": "state_change",
  "event_data": {
    "key": "sensor.temperature",
    "old_value": "22.5",
    "new_value": "23.1"
  },
  "severity": "info"
}

# Submit metric
POST /api/v1/metrics
{
  "service": "org.plantd.Broker",
  "metric_name": "message_throughput",
  "metric_value": 1250.5,
  "tags": {
    "endpoint": "frontend",
    "protocol": "zmq"
  },
  "unit": "messages/sec"
}
```

### Log Querying

```bash
# Get recent logs
GET /api/v1/logs?limit=100&service=org.plantd.Broker

# Get logs by time range
GET /api/v1/logs?start=2024-01-01T00:00:00Z&end=2024-01-02T00:00:00Z

# Get logs by level
GET /api/v1/logs?level=error&limit=50

# Search logs by message content
GET /api/v1/logs?search=connection&service=org.plantd.State

# Get aggregated log counts
GET /api/v1/logs/aggregate?interval=1h&start=2024-01-01T00:00:00Z
```

### Event Querying

```bash
# Get events by type
GET /api/v1/events?event_type=state_change&limit=100

# Get events by service
GET /api/v1/events?service=org.plantd.State&start=2024-01-01T00:00:00Z

# Get events by severity
GET /api/v1/events?severity=error&limit=50
```

### Metrics Querying

```bash
# Get metrics by name
GET /api/v1/metrics?metric_name=message_throughput&limit=1000

# Get metrics with aggregation
GET /api/v1/metrics/aggregate?metric_name=cpu_usage&interval=5m&func=avg

# Get metrics by tags
GET /api/v1/metrics?tags={"environment":"production"}
```

## Integration

### Message Bus Integration

The logger service subscribes to message bus topics to automatically collect logs:

```go
// Example log message structure
type LogMessage struct {
    Time      time.Time              `json:"time"`
    Service   string                 `json:"service"`
    Level     string                 `json:"level"`
    Message   string                 `json:"message"`
    Context   map[string]interface{} `json:"context,omitempty"`
    Host      string                 `json:"host,omitempty"`
    PID       int                    `json:"pid,omitempty"`
}
```

### Service Integration

Other plantd services can send logs directly:

```go
import "github.com/geoffjay/plantd/logger/client"

// Create logger client
loggerClient := client.New("http://localhost:8082")

// Send log entry
err := loggerClient.Log(client.LogEntry{
    Service: "org.plantd.MyService",
    Level:   "info",
    Message: "Operation completed successfully",
    Context: map[string]interface{}{
        "operation": "data_processing",
        "duration":  "1.5s",
    },
})
```

## Log Management

### Retention Policies

Configure automatic log cleanup:

```sql
-- Drop logs older than 30 days
SELECT add_retention_policy('logs', INTERVAL '30 days');

-- Drop events older than 90 days
SELECT add_retention_policy('events', INTERVAL '90 days');

-- Drop metrics older than 1 year
SELECT add_retention_policy('metrics', INTERVAL '1 year');
```

### Compression

Enable compression for older data:

```sql
-- Compress logs older than 7 days
SELECT add_compression_policy('logs', INTERVAL '7 days');

-- Compress events older than 14 days
SELECT add_compression_policy('events', INTERVAL '14 days');
```

### Continuous Aggregates

Create materialized views for common queries:

```sql
-- Hourly log counts by service and level
CREATE MATERIALIZED VIEW logs_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    service,
    level,
    COUNT(*) as log_count
FROM logs
GROUP BY hour, service, level;

-- Daily metric averages
CREATE MATERIALIZED VIEW metrics_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS day,
    service,
    metric_name,
    AVG(metric_value) as avg_value,
    MAX(metric_value) as max_value,
    MIN(metric_value) as min_value
FROM metrics
GROUP BY day, service, metric_name;
```

## Monitoring and Alerting

### Health Checks

```bash
# Check logger service health
curl http://localhost:8082/health

# Check database connectivity
curl http://localhost:8082/health/database
```

### Metrics

The logger service exposes its own metrics:

- **Log Ingestion Rate**: Logs received per second
- **Database Performance**: Query response times
- **Storage Usage**: Database size and growth rate
- **Error Rates**: Failed log ingestion attempts

### Alerting

Configure alerts for critical conditions:

```yaml
# alerts.yaml
alerts:
  - name: "high_error_rate"
    condition: "error_logs_per_minute > 100"
    action: "webhook"
    webhook_url: "http://alertmanager:9093/api/v1/alerts"
  
  - name: "database_connection_failed"
    condition: "database_health == false"
    action: "email"
    email: "admin@example.com"
```

## Development

### Hot Reload

```bash
# Install Air for hot reload
go install github.com/cosmtrek/air@latest

# Start with hot reload
air
```

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests (requires TimescaleDB)
go test -tags=integration ./...

# Run with coverage
go test -cover ./...
```

### Database Migrations

```bash
# Run database migrations
make migrate-up

# Rollback migrations
make migrate-down

# Create new migration
make migrate-create name=add_new_table
```

## Deployment

### Docker

```bash
# Build Docker image
docker build -t plantd-logger .

# Run with Docker Compose
docker-compose up logger
```

### Production Configuration

```yaml
# production.yaml
database:
  url: "postgres://plantd:password@db-cluster:5432/plantd_logs?sslmode=require"
  max_connections: 50

server:
  port: 8082
  host: "0.0.0.0"

retention:
  days: 90
  archive_enabled: true
  archive_path: "/data/archive"

log:
  level: "warn"
  format: "json"
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**:
   ```bash
   # Check TimescaleDB extension
   sudo -u postgres psql -c "SELECT * FROM pg_extension WHERE extname='timescaledb';"
   
   # Verify connection string
   PLANTD_LOGGER_LOG_LEVEL=debug ./build/plantd-logger
   ```

2. **High Memory Usage**:
   ```bash
   # Adjust batch size for log ingestion
   export PLANTD_LOGGER_BATCH_SIZE="100"
   
   # Enable compression
   export PLANTD_LOGGER_COMPRESSION_ENABLED="true"
   ```

3. **Slow Queries**:
   ```sql
   -- Add indexes for common queries
   CREATE INDEX idx_logs_service_time ON logs (service, time DESC);
   CREATE INDEX idx_logs_level_time ON logs (level, time DESC);
   ```

### Performance Tuning

```sql
-- Optimize TimescaleDB settings
ALTER DATABASE plantd_logs SET timescaledb.max_background_workers = 8;
ALTER DATABASE plantd_logs SET shared_preload_libraries = 'timescaledb';

-- Tune chunk intervals
SELECT set_chunk_time_interval('logs', INTERVAL '1 day');
SELECT set_chunk_time_interval('metrics', INTERVAL '6 hours');
```

## Contributing

See the main [plantd contributing guide](../README.md#contributing) for development setup and guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.
