[![Go Report Card](https://goreportcard.com/badge/github.com/geoffjay/plantd/module/metric)](https://goreportcard.com/report/github.com/geoffjay/plantd/module/metric)

---

# 📊 Metric Module

The metric module provides comprehensive metrics collection, processing, and aggregation for the plantd distributed control system. It can operate as both a producer (generating metrics) and consumer (collecting and processing metrics from other services).

## Features

- **Dual Mode Operation**: Functions as both metric producer and consumer
- **Real-time Collection**: Continuous metrics gathering from system components
- **Data Aggregation**: Statistical aggregation and time-series processing
- **Multiple Formats**: Support for various metric formats (Prometheus, InfluxDB, custom)
- **Configurable Storage**: Flexible storage backends and retention policies
- **Alert Generation**: Threshold-based alerting and anomaly detection
- **Performance Monitoring**: System and application performance metrics
- **Custom Metrics**: Support for user-defined metrics and dimensions

## Quick Start

### Prerequisites

- Go 1.24 or later
- ZeroMQ library (for message bus integration)
- Storage backend (InfluxDB, Prometheus, or file-based)

### Installation

```bash
# Build the metric module
cd module/metric
go build -o metric main.go

# Or build from project root
make build-metric
```

### Basic Usage

```bash
# Start as consumer (default)
./metric

# Start as producer
PLANTD_MODULE_METRIC_TYPE=producer ./metric

# Start with debug logging
PLANTD_MODULE_METRIC_LOG_LEVEL=debug ./metric

# Start with custom configuration
PLANTD_MODULE_METRIC_CONFIG=config/metric.yaml ./metric
```

## Configuration

### Environment Variables

```bash
# Service type (consumer or producer)
export PLANTD_MODULE_METRIC_TYPE="consumer"

# Logging configuration
export PLANTD_MODULE_METRIC_LOG_LEVEL="info"
export PLANTD_MODULE_METRIC_LOG_FORMAT="text"

# Message bus configuration
export PLANTD_MODULE_METRIC_BUS_ENDPOINT="tcp://localhost:11001"

# Storage configuration
export PLANTD_MODULE_METRIC_STORAGE_TYPE="influxdb"
export PLANTD_MODULE_METRIC_STORAGE_URL="http://localhost:8086"

# Collection interval
export PLANTD_MODULE_METRIC_INTERVAL="10s"

# Retention policy
export PLANTD_MODULE_METRIC_RETENTION="30d"
```

### Configuration File

```yaml
# config/metric.yaml
service:
  type: "consumer"  # or "producer"
  interval: "10s"
  batch_size: 100

logging:
  level: "info"
  format: "text"

bus:
  endpoint: "tcp://localhost:11001"
  topics:
    - "metrics.*"
    - "system.*"
    - "application.*"

storage:
  type: "influxdb"  # influxdb, prometheus, file
  url: "http://localhost:8086"
  database: "plantd_metrics"
  username: "plantd"
  password: "password"
  retention: "30d"

collection:
  system_metrics: true
  application_metrics: true
  custom_metrics: true
  
  # System metrics to collect
  system:
    - "cpu_usage"
    - "memory_usage"
    - "disk_usage"
    - "network_io"
    - "process_count"
  
  # Application metrics to collect
  application:
    - "request_count"
    - "response_time"
    - "error_rate"
    - "queue_depth"

alerts:
  enabled: true
  rules:
    - name: "high_cpu"
      metric: "cpu_usage"
      threshold: 80
      operator: ">"
      duration: "5m"
    - name: "high_memory"
      metric: "memory_usage"
      threshold: 90
      operator: ">"
      duration: "2m"
```

## Operation Modes

### Consumer Mode

In consumer mode, the metric module collects metrics from other services:

```bash
# Start as consumer
PLANTD_MODULE_METRIC_TYPE=consumer ./metric
```

**Responsibilities:**
- Subscribe to metric topics on the message bus
- Collect metrics from plantd services
- Aggregate and store metrics in configured backend
- Generate alerts based on thresholds
- Provide query interface for stored metrics

### Producer Mode

In producer mode, the metric module generates and publishes metrics:

```bash
# Start as producer
PLANTD_MODULE_METRIC_TYPE=producer ./metric
```

**Responsibilities:**
- Collect system and application metrics
- Generate custom metrics based on configuration
- Publish metrics to the message bus
- Monitor local system resources
- Report service health and performance

## Metric Types

### System Metrics

Automatically collected system-level metrics:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "service": "system",
  "metrics": {
    "cpu_usage_percent": 45.2,
    "memory_usage_bytes": 1073741824,
    "memory_usage_percent": 65.5,
    "disk_usage_bytes": 5368709120,
    "disk_usage_percent": 78.3,
    "network_rx_bytes": 1048576,
    "network_tx_bytes": 2097152,
    "load_average_1m": 1.25,
    "load_average_5m": 1.15,
    "load_average_15m": 1.05
  }
}
```

### Application Metrics

Service-specific performance metrics:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "service": "org.plantd.Broker",
  "metrics": {
    "message_count": 1250,
    "message_rate": 125.5,
    "queue_depth": 45,
    "connection_count": 12,
    "error_count": 2,
    "response_time_ms": 15.3,
    "uptime_seconds": 86400
  }
}
```

### Custom Metrics

User-defined metrics with flexible dimensions:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "service": "org.plantd.TempSensor",
  "metrics": {
    "temperature_celsius": 23.5,
    "humidity_percent": 65.2,
    "pressure_hpa": 1013.25
  },
  "tags": {
    "location": "server_room",
    "sensor_id": "temp_001",
    "environment": "production"
  }
}
```

## Storage Backends

### InfluxDB

Time-series database optimized for metrics:

```yaml
storage:
  type: "influxdb"
  url: "http://localhost:8086"
  database: "plantd_metrics"
  username: "plantd"
  password: "password"
  retention: "30d"
  precision: "s"
```

### Prometheus

Prometheus-compatible metrics storage:

```yaml
storage:
  type: "prometheus"
  url: "http://localhost:9090"
  job_name: "plantd"
  scrape_interval: "15s"
  metrics_path: "/metrics"
```

### File-based Storage

Simple file-based storage for development:

```yaml
storage:
  type: "file"
  path: "/var/lib/plantd/metrics"
  format: "json"  # json, csv, parquet
  rotation: "daily"
  compression: true
```

## API Endpoints

### Metrics Query

```bash
# Get recent metrics
GET /api/v1/metrics?service=org.plantd.Broker&limit=100

# Get metrics by time range
GET /api/v1/metrics?start=2024-01-01T00:00:00Z&end=2024-01-02T00:00:00Z

# Get specific metric
GET /api/v1/metrics?metric=cpu_usage&service=system

# Get aggregated metrics
GET /api/v1/metrics/aggregate?metric=response_time&interval=5m&func=avg
```

### Metrics Ingestion

```bash
# Submit single metric
POST /api/v1/metrics
{
  "service": "org.plantd.MyService",
  "timestamp": "2024-01-01T12:00:00Z",
  "metrics": {
    "custom_metric": 42.5
  },
  "tags": {
    "environment": "production"
  }
}

# Submit batch metrics
POST /api/v1/metrics/batch
{
  "metrics": [
    {
      "service": "service1",
      "timestamp": "2024-01-01T12:00:00Z",
      "metrics": {"value": 1.0}
    },
    {
      "service": "service2", 
      "timestamp": "2024-01-01T12:00:01Z",
      "metrics": {"value": 2.0}
    }
  ]
}
```

### Health and Status

```bash
# Health check
GET /health

# Service status
GET /status
{
  "status": "healthy",
  "mode": "consumer",
  "uptime": "2h30m",
  "metrics_processed": 15420,
  "storage_status": "connected",
  "last_metric": "2024-01-01T12:00:00Z"
}

# Storage status
GET /storage/status
{
  "type": "influxdb",
  "status": "connected",
  "database": "plantd_metrics",
  "retention": "30d",
  "disk_usage": "2.5GB"
}
```

## Message Bus Integration

### Metric Publishing (Producer Mode)

```go
// Example metric publishing
type MetricMessage struct {
    Service   string                 `json:"service"`
    Timestamp time.Time              `json:"timestamp"`
    Metrics   map[string]float64     `json:"metrics"`
    Tags      map[string]string      `json:"tags,omitempty"`
}

func publishMetric(bus *zmq.Socket, metric MetricMessage) error {
    data, err := json.Marshal(metric)
    if err != nil {
        return err
    }
    
    topic := fmt.Sprintf("metrics.%s", metric.Service)
    return bus.SendMessage(topic, data)
}
```

### Metric Consumption (Consumer Mode)

```go
// Example metric consumption
func consumeMetrics(bus *zmq.Socket, storage Storage) {
    for {
        topic, data, err := bus.RecvMessage()
        if err != nil {
            log.Error(err)
            continue
        }
        
        var metric MetricMessage
        if err := json.Unmarshal(data, &metric); err != nil {
            log.Error(err)
            continue
        }
        
        if err := storage.Store(metric); err != nil {
            log.Error(err)
        }
    }
}
```

## Alerting

### Alert Rules

Configure threshold-based alerts:

```yaml
alerts:
  enabled: true
  webhook_url: "http://alertmanager:9093/api/v1/alerts"
  
  rules:
    - name: "high_cpu_usage"
      metric: "cpu_usage_percent"
      threshold: 80
      operator: ">"
      duration: "5m"
      severity: "warning"
      message: "CPU usage is above 80% for 5 minutes"
    
    - name: "critical_memory_usage"
      metric: "memory_usage_percent"
      threshold: 95
      operator: ">"
      duration: "1m"
      severity: "critical"
      message: "Memory usage is critically high"
    
    - name: "service_down"
      metric: "uptime_seconds"
      threshold: 60
      operator: "<"
      duration: "30s"
      severity: "critical"
      message: "Service appears to be down"
```

### Alert Notifications

```bash
# Example alert webhook payload
POST /api/v1/alerts
{
  "alerts": [
    {
      "name": "high_cpu_usage",
      "service": "org.plantd.Broker",
      "severity": "warning",
      "message": "CPU usage is above 80% for 5 minutes",
      "value": 85.2,
      "threshold": 80,
      "timestamp": "2024-01-01T12:00:00Z",
      "tags": {
        "environment": "production",
        "host": "broker-01"
      }
    }
  ]
}
```

## Monitoring and Visualization

### Grafana Integration

Example Grafana dashboard configuration:

```json
{
  "dashboard": {
    "title": "Plantd System Metrics",
    "panels": [
      {
        "title": "CPU Usage",
        "type": "graph",
        "targets": [
          {
            "query": "SELECT mean(cpu_usage_percent) FROM system_metrics WHERE time >= now() - 1h GROUP BY time(1m)"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph", 
        "targets": [
          {
            "query": "SELECT mean(memory_usage_percent) FROM system_metrics WHERE time >= now() - 1h GROUP BY time(1m)"
          }
        ]
      }
    ]
  }
}
```

### Prometheus Integration

Export metrics in Prometheus format:

```bash
# Prometheus metrics endpoint
GET /metrics

# Example output:
# HELP plantd_cpu_usage_percent CPU usage percentage
# TYPE plantd_cpu_usage_percent gauge
plantd_cpu_usage_percent{service="system",host="server-01"} 45.2

# HELP plantd_memory_usage_bytes Memory usage in bytes
# TYPE plantd_memory_usage_bytes gauge
plantd_memory_usage_bytes{service="system",host="server-01"} 1073741824

# HELP plantd_message_count_total Total number of messages processed
# TYPE plantd_message_count_total counter
plantd_message_count_total{service="org.plantd.Broker"} 1250
```

## Development

### Custom Metric Collectors

```go
// Example custom metric collector
type CustomCollector struct {
    name     string
    interval time.Duration
}

func (c *CustomCollector) Collect() (map[string]float64, error) {
    metrics := make(map[string]float64)
    
    // Collect custom metrics
    metrics["custom_value"] = getCustomValue()
    metrics["business_metric"] = getBusinessMetric()
    
    return metrics, nil
}

func (c *CustomCollector) Start(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            metrics, err := c.Collect()
            if err != nil {
                log.Error(err)
                continue
            }
            
            publishMetrics(c.name, metrics)
            
        case <-ctx.Done():
            return
        }
    }
}
```

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Benchmark tests
go test -bench=. ./...

# Test with different storage backends
STORAGE_TYPE=influxdb go test ./storage/...
STORAGE_TYPE=prometheus go test ./storage/...
```

### Docker

```bash
# Build Docker image
docker build -t plantd-metric .

# Run as consumer
docker run -e PLANTD_MODULE_METRIC_TYPE=consumer plantd-metric

# Run as producer
docker run -e PLANTD_MODULE_METRIC_TYPE=producer plantd-metric
```

## Deployment

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  metric-consumer:
    build: .
    environment:
      - PLANTD_MODULE_METRIC_TYPE=consumer
      - PLANTD_MODULE_METRIC_STORAGE_URL=http://influxdb:8086
    depends_on:
      - influxdb
  
  metric-producer:
    build: .
    environment:
      - PLANTD_MODULE_METRIC_TYPE=producer
      - PLANTD_MODULE_METRIC_INTERVAL=30s
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
  
  influxdb:
    image: influxdb:1.8
    environment:
      - INFLUXDB_DB=plantd_metrics
      - INFLUXDB_USER=plantd
      - INFLUXDB_USER_PASSWORD=password
    ports:
      - "8086:8086"
```

### Kubernetes

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: plantd-metric-consumer
spec:
  replicas: 2
  selector:
    matchLabels:
      app: plantd-metric-consumer
  template:
    metadata:
      labels:
        app: plantd-metric-consumer
    spec:
      containers:
      - name: metric
        image: plantd-metric:latest
        env:
        - name: PLANTD_MODULE_METRIC_TYPE
          value: "consumer"
        - name: PLANTD_MODULE_METRIC_STORAGE_URL
          value: "http://influxdb:8086"
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: plantd-metric-producer
spec:
  selector:
    matchLabels:
      app: plantd-metric-producer
  template:
    metadata:
      labels:
        app: plantd-metric-producer
    spec:
      hostNetwork: true
      hostPID: true
      containers:
      - name: metric
        image: plantd-metric:latest
        env:
        - name: PLANTD_MODULE_METRIC_TYPE
          value: "producer"
        volumeMounts:
        - name: proc
          mountPath: /host/proc
          readOnly: true
        - name: sys
          mountPath: /host/sys
          readOnly: true
      volumes:
      - name: proc
        hostPath:
          path: /proc
      - name: sys
        hostPath:
          path: /sys
```

## Troubleshooting

### Common Issues

1. **Storage Connection Failed**:
   ```bash
   # Check storage backend connectivity
   curl http://localhost:8086/ping  # InfluxDB
   curl http://localhost:9090/-/healthy  # Prometheus
   
   # Verify credentials
   PLANTD_MODULE_METRIC_LOG_LEVEL=debug ./metric
   ```

2. **High Memory Usage**:
   ```bash
   # Reduce batch size
   export PLANTD_MODULE_METRIC_BATCH_SIZE=50
   
   # Increase flush interval
   export PLANTD_MODULE_METRIC_FLUSH_INTERVAL=30s
   ```

3. **Missing Metrics**:
   ```bash
   # Check message bus connectivity
   export PLANTD_MODULE_METRIC_LOG_LEVEL=trace
   
   # Verify topic subscriptions
   ./metric --list-topics
   ```

### Performance Tuning

```yaml
# Optimize for high-throughput scenarios
performance:
  batch_size: 1000
  flush_interval: "10s"
  worker_count: 4
  buffer_size: 10000
  compression: true
```

## Contributing

See the main [plantd contributing guide](../../README.md#contributing) for development setup and guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details. 
