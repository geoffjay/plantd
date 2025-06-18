/**
 * Services Management JavaScript
 * Handles service control, filtering, and real-time updates
 */

class ServicesManager {
    constructor() {
        this.currentFilter = 'all';
        this.currentSort = 'name';
        this.services = [];
        this.currentAction = null;
        this.currentService = null;
        
        this.init();
    }

    init() {
        console.log('Initializing services manager...');
        this.setupEventListeners();
        this.loadServices();
        
        // Auto-refresh every 30 seconds
        setInterval(() => {
            this.loadServices();
        }, 30000);
    }

    setupEventListeners() {
        // Filter change handler
        const filterSelect = document.getElementById('filter');
        if (filterSelect) {
            filterSelect.addEventListener('change', (e) => {
                this.currentFilter = e.target.value;
                this.loadServices();
            });
        }

        // Sort change handler
        const sortSelect = document.getElementById('sort');
        if (sortSelect) {
            sortSelect.addEventListener('change', (e) => {
                this.currentSort = e.target.value;
                this.loadServices();
            });
        }

        // Close modal when clicking outside
        window.addEventListener('click', (e) => {
            if (e.target.classList.contains('service-menu-backdrop')) {
                this.hideAllMenus();
            }
        });

        // Close modal with Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeServiceModal();
                this.hideAllMenus();
            }
        });
    }

    loadServices() {
        console.log('Loading services...');
        
        const params = new URLSearchParams({
            filter: this.currentFilter,
            sort: this.currentSort
        });

        fetch(`/api/services?${params}`)
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    this.services = data.data.services;
                    this.updateServicesDisplay(data.data);
                } else {
                    console.error('Failed to load services:', data.error);
                    this.showError('Failed to load services: ' + data.error);
                }
            })
            .catch(error => {
                console.error('Error loading services:', error);
                this.showError('Error loading services: ' + error.message);
            });
    }

    updateServicesDisplay(data) {
        // Update service count
        const countElement = document.querySelector('.text-sm.text-gray-500');
        if (countElement && countElement.textContent.includes('Total:')) {
            countElement.textContent = `Total: ${data.count} services`;
        }

        // In a real implementation, you would update the services list here
        // For now, we'll just log the data
        console.log('Services updated:', data);
    }

    refreshServices() {
        console.log('Manual refresh requested');
        this.loadServices();
    }

    toggleServiceMenu(button) {
        const serviceIndex = button.getAttribute('data-service-index');
        const menuId = `service-menu-${serviceIndex}`;
        const menu = document.getElementById(menuId);
        
        if (!menu) return;

        // Hide all other menus first
        this.hideAllMenus();

        // Toggle current menu
        if (menu.classList.contains('hidden')) {
            menu.classList.remove('hidden');
            
            // Add click handlers to menu items
            this.setupMenuHandlers(menu, serviceIndex);
        } else {
            menu.classList.add('hidden');
        }
    }

    hideAllMenus() {
        const menus = document.querySelectorAll('[id^="service-menu-"]');
        menus.forEach(menu => {
            menu.classList.add('hidden');
        });
    }

    setupMenuHandlers(menu, serviceIndex) {
        const menuItems = menu.querySelectorAll('a');
        
        menuItems.forEach(item => {
            item.onclick = (e) => {
                e.preventDefault();
                const action = item.textContent.toLowerCase().trim();
                this.handleServiceAction(action, serviceIndex);
                this.hideAllMenus();
            };
        });
    }

    handleServiceAction(action, serviceIndex) {
        console.log(`Service action: ${action} for service ${serviceIndex}`);
        
        const serviceName = this.getServiceName(serviceIndex);
        if (!serviceName) {
            this.showError('Unable to identify service');
            return;
        }

        this.currentAction = action;
        this.currentService = serviceName;

        switch (action) {
            case 'view details':
                window.location.href = `/services/${serviceName}`;
                break;
            case 'restart':
                this.showServiceModal('Restart Service', `Are you sure you want to restart ${serviceName}?`, 'warning');
                break;
            case 'stop':
                this.showServiceModal('Stop Service', `Are you sure you want to stop ${serviceName}?`, 'danger');
                break;
            case 'scale':
                this.showScaleModal(serviceName);
                break;
            default:
                console.warn('Unknown action:', action);
        }
    }

    getServiceName(serviceIndex) {
        // In a real implementation, you would get the actual service name
        // For now, return a placeholder
        return `service-${serviceIndex}`;
    }

    showServiceModal(title, message, type = 'info') {
        const modal = document.getElementById('service-action-modal');
        const titleElement = document.getElementById('modal-title');
        const contentElement = document.getElementById('modal-content');
        const confirmButton = document.getElementById('modal-confirm');

        if (!modal || !titleElement || !contentElement || !confirmButton) {
            console.error('Modal elements not found');
            return;
        }

        titleElement.textContent = title;
        contentElement.innerHTML = `<p class="text-sm text-gray-600">${message}</p>`;

        // Style confirm button based on action type
        confirmButton.className = this.getButtonClass(type);
        confirmButton.textContent = this.getButtonText(this.currentAction);

        modal.classList.remove('hidden');
    }

    showScaleModal(serviceName) {
        const modal = document.getElementById('service-action-modal');
        const titleElement = document.getElementById('modal-title');
        const contentElement = document.getElementById('modal-content');
        const confirmButton = document.getElementById('modal-confirm');

        titleElement.textContent = 'Scale Service';
        contentElement.innerHTML = `
            <div class="space-y-3">
                <p class="text-sm text-gray-600">Scale ${serviceName} workers:</p>
                <div>
                    <label for="worker-count" class="block text-sm font-medium text-gray-700 mb-1">Number of workers</label>
                    <input type="number" id="worker-count" min="0" max="10" value="3" 
                           class="border border-gray-300 rounded-md px-3 py-2 w-full focus:outline-none focus:ring-2 focus:ring-blue-500">
                </div>
            </div>
        `;

        confirmButton.className = 'px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md';
        confirmButton.textContent = 'Scale';

        modal.classList.remove('hidden');
    }

    getButtonClass(type) {
        const classes = {
            'info': 'px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md',
            'warning': 'px-4 py-2 text-sm font-medium text-white bg-yellow-600 hover:bg-yellow-700 rounded-md',
            'danger': 'px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md'
        };
        return classes[type] || classes.info;
    }

    getButtonText(action) {
        const texts = {
            'restart': 'Restart',
            'stop': 'Stop',
            'start': 'Start',
            'scale': 'Scale'
        };
        return texts[action] || 'Confirm';
    }

    closeServiceModal() {
        const modal = document.getElementById('service-action-modal');
        if (modal) {
            modal.classList.add('hidden');
        }
        this.currentAction = null;
        this.currentService = null;
    }

    confirmServiceAction() {
        if (!this.currentAction || !this.currentService) {
            console.error('No current action or service');
            return;
        }

        console.log(`Confirming action: ${this.currentAction} for service: ${this.currentService}`);

        let endpoint = '';
        let method = 'POST';
        let body = null;

        switch (this.currentAction) {
            case 'restart':
                endpoint = `/api/services/${this.currentService}/restart`;
                break;
            case 'stop':
                endpoint = `/api/services/${this.currentService}/stop`;
                break;
            case 'start':
                endpoint = `/api/services/${this.currentService}/start`;
                break;
            case 'scale':
                const workerCount = document.getElementById('worker-count')?.value;
                if (!workerCount) {
                    this.showError('Please specify worker count');
                    return;
                }
                endpoint = `/api/services/${this.currentService}/scale`;
                body = new FormData();
                body.append('workers', workerCount);
                break;
            default:
                console.error('Unknown action:', this.currentAction);
                return;
        }

        // Show loading state
        const confirmButton = document.getElementById('modal-confirm');
        if (confirmButton) {
            confirmButton.disabled = true;
            confirmButton.textContent = 'Processing...';
        }

        fetch(endpoint, {
            method: method,
            body: body
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                this.showSuccess(data.message || 'Action completed successfully');
                this.closeServiceModal();
                // Refresh services list after a short delay
                setTimeout(() => {
                    this.loadServices();
                }, 1000);
            } else {
                this.showError(data.error || 'Action failed');
            }
        })
        .catch(error => {
            console.error('Error performing action:', error);
            this.showError('Error performing action: ' + error.message);
        })
        .finally(() => {
            // Reset button state
            if (confirmButton) {
                confirmButton.disabled = false;
                confirmButton.textContent = this.getButtonText(this.currentAction);
            }
        });
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `fixed top-4 right-4 z-50 p-4 rounded-md shadow-lg max-w-sm ${this.getNotificationClass(type)}`;
        notification.innerHTML = `
            <div class="flex items-center">
                <div class="flex-shrink-0">
                    ${this.getNotificationIcon(type)}
                </div>
                <div class="ml-3">
                    <p class="text-sm font-medium">${message}</p>
                </div>
                <div class="ml-auto pl-3">
                    <button onclick="this.parentElement.parentElement.parentElement.remove()" class="text-sm underline">
                        ×
                    </button>
                </div>
            </div>
        `;

        document.body.appendChild(notification);

        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 5000);
    }

    getNotificationClass(type) {
        const classes = {
            'success': 'bg-green-100 border border-green-400 text-green-700',
            'error': 'bg-red-100 border border-red-400 text-red-700',
            'info': 'bg-blue-100 border border-blue-400 text-blue-700'
        };
        return classes[type] || classes.info;
    }

    getNotificationIcon(type) {
        const icons = {
            'success': '<svg class="w-5 h-5 text-green-400" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path></svg>',
            'error': '<svg class="w-5 h-5 text-red-400" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"></path></svg>',
            'info': '<svg class="w-5 h-5 text-blue-400" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"></path></svg>'
        };
        return icons[type] || icons.info;
    }
}

// Global functions for template usage
function refreshServices() {
    if (window.servicesManager) {
        window.servicesManager.refreshServices();
    }
}

function toggleServiceMenu(button) {
    if (window.servicesManager) {
        window.servicesManager.toggleServiceMenu(button);
    }
}

function closeServiceModal() {
    if (window.servicesManager) {
        window.servicesManager.closeServiceModal();
    }
}

function confirmServiceAction() {
    if (window.servicesManager) {
        window.servicesManager.confirmServiceAction();
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded, initializing services manager');
    window.servicesManager = new ServicesManager();
});

// Export for manual use
window.ServicesManager = ServicesManager; 
