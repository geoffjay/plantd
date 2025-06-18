/**
 * Dashboard Real-time Updates
 * Handles Server-Sent Events for live dashboard data updates
 */

class DashboardUpdater {
    constructor() {
        this.eventSource = null;
        this.statusEventSource = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000; // Start with 1 second
        this.isConnected = false;
        
        this.init();
    }

    init() {
        console.log('Initializing dashboard updater...');
        this.setupSSEConnection();
        this.setupStatusConnection();
        this.setupEventListeners();
    }

    setupSSEConnection() {
        if (this.eventSource) {
            this.eventSource.close();
        }

        console.log('Connecting to dashboard SSE stream...');
        this.eventSource = new EventSource('/dashboard/sse');

        this.eventSource.onopen = () => {
            console.log('Dashboard SSE connection opened');
            this.isConnected = true;
            this.reconnectAttempts = 0;
            this.reconnectDelay = 1000;
            this.updateConnectionStatus(true);
        };

        this.eventSource.addEventListener('dashboard-update', (event) => {
            try {
                const data = JSON.parse(event.data);
                this.updateDashboard(data);
            } catch (error) {
                console.error('Error parsing dashboard update:', error);
            }
        });

        this.eventSource.onerror = (error) => {
            console.error('Dashboard SSE error:', error);
            this.isConnected = false;
            this.updateConnectionStatus(false);
            this.handleConnectionError();
        };
    }

    setupStatusConnection() {
        if (this.statusEventSource) {
            this.statusEventSource.close();
        }

        console.log('Connecting to status SSE stream...');
        this.statusEventSource = new EventSource('/system/status/sse');

        this.statusEventSource.addEventListener('status-update', (event) => {
            try {
                const data = JSON.parse(event.data);
                this.updateSystemStatus(data);
            } catch (error) {
                console.error('Error parsing status update:', error);
            }
        });

        this.statusEventSource.onerror = (error) => {
            console.error('Status SSE error:', error);
        };
    }

    setupEventListeners() {
        // Handle page visibility changes to reconnect when tab becomes visible
        document.addEventListener('visibilitychange', () => {
            if (!document.hidden && !this.isConnected) {
                console.log('Tab became visible, attempting to reconnect...');
                this.setupSSEConnection();
                this.setupStatusConnection();
            }
        });

        // Handle window focus/blur
        window.addEventListener('focus', () => {
            if (!this.isConnected) {
                this.setupSSEConnection();
                this.setupStatusConnection();
            }
        });
    }

    updateDashboard(data) {
        console.log('Updating dashboard with data:', data);
        
        // Update timestamp
        if (data.timestamp) {
            this.updateLastUpdated(new Date(data.timestamp));
        }

        // Update system health
        if (data.system_health) {
            this.updateSystemHealth(data.system_health);
        }

        // Update services
        if (data.services) {
            this.updateServicesOverview(data.services);
        }

        // Update metrics
        if (data.metrics) {
            this.updateMetrics(data.metrics);
        }
    }

    updateSystemHealth(health) {
        // Update health status
        const statusElement = document.getElementById('health-status');
        if (statusElement) {
            statusElement.textContent = this.formatHealthStatus(health.status);
            statusElement.className = `inline-flex px-2 py-1 text-xs font-semibold rounded-full ${this.getHealthStatusClass(health.status)}`;
        }

        // Update uptime
        const uptimeElement = document.getElementById('system-uptime');
        if (uptimeElement) {
            uptimeElement.textContent = health.uptime || 'Unknown';
        }

        // Update components count
        const componentsElement = document.getElementById('health-components');
        if (componentsElement) {
            componentsElement.textContent = health.components || 0;
        }
    }

    updateServicesOverview(services) {
        // Update service count
        const countElement = document.getElementById('service-count');
        if (countElement) {
            countElement.textContent = services.count || 0;
        }

        // Update healthy services
        const healthyElement = document.getElementById('healthy-services');
        if (healthyElement) {
            healthyElement.textContent = services.healthy || 0;
        }

        // Update service health percentage
        const percentage = services.count > 0 ? Math.round((services.healthy / services.count) * 100) : 0;
        const percentageElement = document.getElementById('service-health-percentage');
        if (percentageElement) {
            percentageElement.textContent = `${percentage}%`;
        }
    }

    updateMetrics(metrics) {
        // Update request rate
        const requestRateElement = document.getElementById('request-rate');
        if (requestRateElement) {
            requestRateElement.textContent = this.formatRequestRate(metrics.request_rate);
        }

        // Update response time
        const responseTimeElement = document.getElementById('response-time');
        if (responseTimeElement) {
            responseTimeElement.textContent = `${metrics.response_time_ms}ms`;
        }

        // Update error rate
        const errorRateElement = document.getElementById('error-rate');
        if (errorRateElement) {
            errorRateElement.textContent = `${(metrics.error_rate * 100).toFixed(1)}%`;
        }

        // Update memory usage
        const memoryElement = document.getElementById('memory-usage');
        if (memoryElement) {
            memoryElement.textContent = `${metrics.memory_mb.toFixed(1)} MB`;
        }

        // Update CPU usage
        const cpuElement = document.getElementById('cpu-usage');
        if (cpuElement) {
            cpuElement.textContent = `${metrics.cpu_percent.toFixed(1)}%`;
        }
    }

    updateSystemStatus(status) {
        // This is for quick status updates (every 2 seconds)
        const quickStatusElement = document.getElementById('quick-status');
        if (quickStatusElement) {
            quickStatusElement.textContent = this.formatHealthStatus(status.status);
            quickStatusElement.className = `quick-status ${this.getHealthStatusClass(status.status)}`;
        }
    }

    updateConnectionStatus(connected) {
        const indicator = document.getElementById('connection-status');
        if (indicator) {
            if (connected) {
                indicator.className = 'connection-indicator connected';
                indicator.textContent = 'Live';
                indicator.title = 'Real-time updates active';
            } else {
                indicator.className = 'connection-indicator disconnected';
                indicator.textContent = 'Offline';
                indicator.title = 'Real-time updates unavailable';
            }
        }
    }

    updateLastUpdated(timestamp) {
        const element = document.getElementById('last-updated');
        if (element) {
            element.textContent = `Last updated: ${timestamp.toLocaleTimeString()}`;
        }
    }

    handleConnectionError() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            console.log(`Attempting to reconnect in ${this.reconnectDelay}ms (attempt ${this.reconnectAttempts + 1})`);
            
            setTimeout(() => {
                this.reconnectAttempts++;
                this.reconnectDelay *= 2; // Exponential backoff
                this.setupSSEConnection();
                this.setupStatusConnection();
            }, this.reconnectDelay);
        } else {
            console.log('Max reconnection attempts reached');
            this.updateConnectionStatus(false);
        }
    }

    formatHealthStatus(status) {
        const statusMap = {
            'healthy': 'Healthy',
            'degraded': 'Degraded',
            'unhealthy': 'Unhealthy',
            'unknown': 'Unknown'
        };
        return statusMap[status] || status;
    }

    getHealthStatusClass(status) {
        const classMap = {
            'healthy': 'bg-green-100 text-green-800',
            'degraded': 'bg-yellow-100 text-yellow-800',
            'unhealthy': 'bg-red-100 text-red-800',
            'unknown': 'bg-gray-100 text-gray-800'
        };
        return classMap[status] || 'bg-gray-100 text-gray-800';
    }

    formatRequestRate(rate) {
        if (rate < 1) {
            return '< 1/sec';
        }
        return `${rate.toFixed(1)}/sec`;
    }

    // Public methods for manual updates
    refresh() {
        console.log('Manual refresh requested');
        fetch('/api/dashboard/data')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    this.updateDashboard({
                        timestamp: new Date().toISOString(),
                        system_health: data.data.system_health ? {
                            status: data.data.health_status,
                            uptime: data.data.uptime,
                            components: Object.keys(data.data.health_components || {}).length
                        } : null,
                        services: {
                            count: data.data.service_count,
                            healthy: data.data.service_count // Simplified for now
                        },
                        metrics: data.data.performance_data ? {
                            request_rate: data.data.performance_data.request_rate || 0,
                            response_time_ms: data.data.performance_data.response_time || 0,
                            error_rate: data.data.performance_data.error_rate || 0,
                            memory_mb: 0, // Would need system metrics
                            cpu_percent: 0
                        } : null
                    });
                }
            })
            .catch(error => {
                console.error('Failed to refresh dashboard data:', error);
            });
    }

    disconnect() {
        console.log('Disconnecting dashboard updater');
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
        if (this.statusEventSource) {
            this.statusEventSource.close();
            this.statusEventSource = null;
        }
        this.isConnected = false;
        this.updateConnectionStatus(false);
    }
}

// CSS for connection status indicator
const connectionCSS = `
.connection-indicator {
    position: fixed;
    top: 20px;
    right: 20px;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
    z-index: 1000;
}

.connection-indicator.connected {
    background-color: #10b981;
    color: white;
}

.connection-indicator.disconnected {
    background-color: #ef4444;
    color: white;
}

.quick-status {
    font-weight: 600;
}
`;

// Inject CSS
const style = document.createElement('style');
style.textContent = connectionCSS;
document.head.appendChild(style);

// Initialize dashboard updater when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded, initializing dashboard updater');
    window.dashboardUpdater = new DashboardUpdater();
});

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (window.dashboardUpdater) {
        window.dashboardUpdater.disconnect();
    }
});

// Export for manual use
window.DashboardUpdater = DashboardUpdater; 
