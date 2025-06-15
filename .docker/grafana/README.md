# PlantD Grafana Dashboards

This directory contains pre-configured Grafana dashboards that are automatically loaded when the development stack starts up.

## Available Dashboards

### 1. PlantD - Logs Overview (`plantd-logs.json`)
- **Purpose**: Comprehensive log analysis and monitoring
- **Features**:
  - Log rate by service (time series)
  - Log rate by level (ERROR, WARN, INFO, DEBUG)
  - Recent logs table with JSON parsing
  - Error logs filtered view
  - ZeroMQ/Message specific logs
  - Service filter template variable

### 2. PlantD - System Metrics (`plantd-system-metrics.json`)  
- **Purpose**: Database and system performance monitoring
- **Features**:
  - Active database connections
  - Total database connections gauge
  - Database table statistics
  - Database activity statistics (transactions, tuples, blocks)
  - Database size growth over time

### 3. PlantD - Service Health (`plantd-service-health.json`)
- **Purpose**: Service status and health monitoring  
- **Features**:
  - Individual service status indicators (Broker, Identity, Proxy, Logger)
  - Error rate by service
  - Service startup events
  - Service lifecycle events table

## Dashboard Access

Once the stack is running, access Grafana at: http://localhost:3333

The dashboards will be automatically available in the Grafana UI without requiring manual import.

## Customization

- Dashboards are editable through the Grafana UI
- Changes made in the UI can be exported and saved back to the JSON files
- New dashboards can be added by placing JSON files in the `dashboards/` directory
- Dashboard provisioning configuration is in `provisioning/dashboards/dashboards.yml`

## Expected Log Format

The dashboards assume logs are structured with at least these fields:
- `service`: Service name (e.g., "plantd-broker", "plantd-identity")  
- `level`: Log level (ERROR, WARN, INFO, DEBUG)
- JSON structured logs are preferred for better parsing and filtering

## Troubleshooting

- If dashboards don't appear, check the Grafana logs for provisioning errors
- Ensure the service names in your logs match the patterns used in the dashboard queries
- Verify that Loki is receiving logs from your services
- Check that the datasource UIDs match between the provisioning and dashboard files 