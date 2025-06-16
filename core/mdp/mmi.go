package mdp

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// MMIHandler handles Majordomo Management Interface requests
type MMIHandler struct {
	broker *Broker
}

// NewMMIHandler creates a new MMI handler for the given broker
func NewMMIHandler(broker *Broker) *MMIHandler {
	return &MMIHandler{broker: broker}
}

// HandleRequest processes an MMI request and returns the appropriate response
func (m *MMIHandler) HandleRequest(service string, request []string) ([]string, error) {
	log.WithFields(log.Fields{
		"service": service,
		"request": request,
	}).Debug("handling MMI request")

	switch service {
	case MMIService:
		return m.handleServiceQuery(request)
	case MMIWorkers:
		return m.handleWorkersQuery(request)
	case MMIHeartbeat:
		return m.handleHeartbeatQuery(request)
	case MMIBroker:
		return m.handleBrokerQuery(request)
	default:
		log.WithField("service", service).Warn("unknown MMI service requested")
		return []string{MMICodeNotImplemented}, nil
	}
}

// IsMMIService checks if a service name is an MMI service
func IsMMIService(serviceName string) bool {
	return strings.HasPrefix(serviceName, MMINamespace)
}

// handleServiceQuery implements mmi.service - checks if a service is available
func (m *MMIHandler) handleServiceQuery(request []string) ([]string, error) {
	if len(request) < 1 {
		return []string{MMICodeError, "service name required"}, nil
	}

	serviceName := request[0]

	// Check if it's an MMI service
	if IsMMIService(serviceName) {
		if _, exists := MMIServices[serviceName]; exists {
			return []string{MMICodeOK}, nil
		}
		return []string{MMICodeNotFound}, nil
	}

	// Check regular services in the broker
	if m.broker != nil && m.broker.services != nil {
		service, exists := m.broker.services[serviceName]
		if !exists {
			return []string{MMICodeNotFound}, nil
		}

		// Service exists, check if it has available workers
		if len(service.waiting) > 0 {
			return []string{MMICodeOK}, nil
		}

		// Service exists but no workers available
		return []string{MMICodeNotFound}, nil
	}

	return []string{MMICodeNotFound}, nil
}

// handleWorkersQuery implements mmi.workers - returns worker count for a service
func (m *MMIHandler) handleWorkersQuery(request []string) ([]string, error) {
	if len(request) < 1 {
		return []string{MMICodeError, "service name required"}, nil
	}

	serviceName := request[0]

	// MMI services don't have workers in the traditional sense
	if IsMMIService(serviceName) {
		if _, exists := MMIServices[serviceName]; exists {
			return []string{MMICodeOK, "1"}, nil // MMI is always available
		}
		return []string{MMICodeNotFound, "0"}, nil
	}

	// Check regular services
	if m.broker != nil && m.broker.services != nil {
		service, exists := m.broker.services[serviceName]
		if !exists {
			return []string{MMICodeNotFound, "0"}, nil
		}

		workerCount := len(service.waiting) + len(service.requests)
		return []string{MMICodeOK, fmt.Sprintf("%d", workerCount)}, nil
	}

	return []string{MMICodeNotFound, "0"}, nil
}

// handleHeartbeatQuery implements mmi.heartbeat - echo service
func (m *MMIHandler) handleHeartbeatQuery(request []string) ([]string, error) {
	// Echo back the request with a timestamp
	response := []string{MMICodeOK, fmt.Sprintf("heartbeat-echo-%d", time.Now().Unix())}

	// Include any request data in the echo
	if len(request) > 0 {
		response = append(response, request...)
	}

	return response, nil
}

// handleBrokerQuery implements mmi.broker - returns broker information
func (m *MMIHandler) handleBrokerQuery(_ []string) ([]string, error) {
	response := []string{MMICodeOK}

	// Add broker information
	info := []string{
		fmt.Sprintf("version=%s/%s", MdpcClient, MdpwWorker),
		fmt.Sprintf("uptime=%d", time.Now().Unix()), // Simple uptime simulation
		fmt.Sprintf("go_version=%s", runtime.Version()),
		fmt.Sprintf("go_arch=%s", runtime.GOARCH),
		fmt.Sprintf("go_os=%s", runtime.GOOS),
	}

	// Add service count if broker is available
	if m.broker != nil && m.broker.services != nil {
		info = append(info, fmt.Sprintf("services=%d", len(m.broker.services)))

		// Add total worker count
		totalWorkers := 0
		for _, service := range m.broker.services {
			totalWorkers += len(service.waiting) + len(service.requests)
		}
		info = append(info, fmt.Sprintf("workers=%d", totalWorkers))
	}

	response = append(response, info...)
	return response, nil
}

// GetSupportedServices returns a list of all supported MMI services
func (m *MMIHandler) GetSupportedServices() []string {
	services := make([]string, 0, len(MMIServices))
	for service := range MMIServices {
		services = append(services, service)
	}
	return services
}

// ValidateMMIRequest validates an MMI request format
func ValidateMMIRequest(service string, request []string) error {
	if !IsMMIService(service) {
		return NewInvalidServiceError(fmt.Sprintf("'%s' is not an MMI service", service), nil)
	}

	if _, exists := MMIServices[service]; !exists {
		return NewServiceNotFoundError(service, nil)
	}

	// Service-specific validation
	switch service {
	case MMIService, MMIWorkers:
		if len(request) < 1 {
			return NewInvalidMessageError("service name required for "+service, nil)
		}
		if request[0] == "" {
			return NewInvalidMessageError("service name cannot be empty", nil)
		}
	case MMIHeartbeat, MMIBroker:
		// These services accept any request format
	}

	return nil
}
