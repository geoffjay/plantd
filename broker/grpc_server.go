package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	brokerv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/broker/v1"
	healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"

	log "github.com/sirupsen/logrus"
)

// BrokerGRPCServer implements the gRPC BrokerService
type BrokerGRPCServer struct {
	mu           sync.RWMutex
	services     map[string]*ServiceRegistration // service_id -> registration
	buses        map[string]*BusRegistration     // bus_name -> bus info
	startTime    time.Time
	lastActivity time.Time
}

// ServiceRegistration holds service registration data
type ServiceRegistration struct {
	ServiceID      string
	RegistrationID string
	Endpoint       string
	Capabilities   []string
	Metadata       map[string]string
	RegisteredAt   time.Time
	LastHeartbeat  time.Time
	Status         brokerv1.ServiceStatus_Status
}

// BusRegistration holds bus information
type BusRegistration struct {
	BusID           string
	Name            string
	Type            brokerv1.BusType
	Config          map[string]string
	SubscriberCount int32
	PublisherCount  int32
	CreatedAt       time.Time
}

// NewBrokerGRPCServer creates a new broker gRPC server
func NewBrokerGRPCServer() *BrokerGRPCServer {
	server := &BrokerGRPCServer{
		services:     make(map[string]*ServiceRegistration),
		buses:        make(map[string]*BusRegistration),
		startTime:    time.Now(),
		lastActivity: time.Now(),
	}

	// Initialize default buses from config
	server.initializeBuses()

	return server
}

// initializeBuses creates default buses from configuration
func (s *BrokerGRPCServer) initializeBuses() {
	config := GetConfig()

	for _, busConfig := range config.Buses {
		busType := getBusType(busConfig.Name)
		s.buses[busConfig.Name] = &BusRegistration{
			BusID: fmt.Sprintf("bus-%s-%d", busConfig.Name, time.Now().Unix()),
			Name:  busConfig.Name,
			Type:  busType,
			Config: map[string]string{
				"frontend": busConfig.Frontend,
				"backend":  busConfig.Backend,
				"capture":  busConfig.Capture,
			},
			SubscriberCount: 0,
			PublisherCount:  0,
			CreatedAt:       time.Now(),
		}
	}
}

// getBusType maps bus name to bus type enum
func getBusType(name string) brokerv1.BusType {
	switch strings.ToLower(name) {
	case "state":
		return brokerv1.BusType_BUS_TYPE_STATE
	case "event":
		return brokerv1.BusType_BUS_TYPE_EVENT
	case "metric":
		return brokerv1.BusType_BUS_TYPE_METRIC
	case "control":
		return brokerv1.BusType_BUS_TYPE_CONTROL
	default:
		return brokerv1.BusType_BUS_TYPE_UNKNOWN
	}
}

// RegisterService registers a new service with the broker
func (s *BrokerGRPCServer) RegisterService(ctx context.Context, req *connect.Request[brokerv1.RegisterServiceRequest]) (*connect.Response[brokerv1.RegisterServiceResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.WithFields(log.Fields{
		"service_id": req.Msg.ServiceId,
		"endpoint":   req.Msg.Endpoint,
	}).Info("Registering service")

	registrationID := fmt.Sprintf("reg-%s-%d", req.Msg.ServiceId, time.Now().Unix())
	now := time.Now()

	registration := &ServiceRegistration{
		ServiceID:      req.Msg.ServiceId,
		RegistrationID: registrationID,
		Endpoint:       req.Msg.Endpoint,
		Capabilities:   req.Msg.Capabilities,
		Metadata:       req.Msg.Metadata,
		RegisteredAt:   now,
		LastHeartbeat:  now,
		Status:         brokerv1.ServiceStatus_HEALTHY,
	}

	s.services[req.Msg.ServiceId] = registration
	s.lastActivity = now

	response := &brokerv1.RegisterServiceResponse{
		RegistrationId: registrationID,
		RegisteredAt:   timestamppb.New(now),
	}

	return connect.NewResponse(response), nil
}

// UnregisterService removes a service registration
func (s *BrokerGRPCServer) UnregisterService(ctx context.Context, req *connect.Request[brokerv1.UnregisterServiceRequest]) (*connect.Response[emptypb.Empty], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.WithFields(log.Fields{
		"service_id":      req.Msg.ServiceId,
		"registration_id": req.Msg.RegistrationId,
	}).Info("Unregistering service")

	if registration, exists := s.services[req.Msg.ServiceId]; exists {
		if registration.RegistrationID == req.Msg.RegistrationId {
			delete(s.services, req.Msg.ServiceId)
			s.lastActivity = time.Now()
		} else {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("invalid registration ID"))
		}
	} else {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("service not found: %s", req.Msg.ServiceId))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetService retrieves service information
func (s *BrokerGRPCServer) GetService(ctx context.Context, req *connect.Request[brokerv1.GetServiceRequest]) (*connect.Response[brokerv1.GetServiceResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	registration, exists := s.services[req.Msg.ServiceId]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("service not found: %s", req.Msg.ServiceId))
	}

	serviceInfo := &brokerv1.ServiceInfo{
		ServiceId:     registration.ServiceID,
		Endpoint:      registration.Endpoint,
		Capabilities:  registration.Capabilities,
		Metadata:      registration.Metadata,
		RegisteredAt:  timestamppb.New(registration.RegisteredAt),
		LastHeartbeat: timestamppb.New(registration.LastHeartbeat),
		Status: &brokerv1.ServiceStatus{
			Status:    registration.Status,
			Message:   "OK",
			Metrics:   registration.Metadata,
			LastCheck: timestamppb.New(registration.LastHeartbeat),
		},
	}

	response := &brokerv1.GetServiceResponse{
		Service: serviceInfo,
	}

	return connect.NewResponse(response), nil
}

// ListServices streams all registered services
func (s *BrokerGRPCServer) ListServices(ctx context.Context, req *connect.Request[brokerv1.ListServicesRequest], stream *connect.ServerStream[brokerv1.ListServicesResponse]) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filter := req.Msg.Filter

	for serviceID, registration := range s.services {
		// Apply filter if provided
		if filter != "" && !strings.Contains(serviceID, filter) {
			continue
		}

		serviceInfo := &brokerv1.ServiceInfo{
			ServiceId:     registration.ServiceID,
			Endpoint:      registration.Endpoint,
			Capabilities:  registration.Capabilities,
			Metadata:      registration.Metadata,
			RegisteredAt:  timestamppb.New(registration.RegisteredAt),
			LastHeartbeat: timestamppb.New(registration.LastHeartbeat),
			Status: &brokerv1.ServiceStatus{
				Status:    registration.Status,
				Message:   "OK",
				Metrics:   registration.Metadata,
				LastCheck: timestamppb.New(registration.LastHeartbeat),
			},
		}

		response := &brokerv1.ListServicesResponse{
			Service: serviceInfo,
		}

		if err := stream.Send(response); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send service: %v", err))
		}
	}

	return nil
}

// RouteMessage routes a message to a target service
func (s *BrokerGRPCServer) RouteMessage(ctx context.Context, req *connect.Request[brokerv1.RouteMessageRequest]) (*connect.Response[brokerv1.RouteMessageResponse], error) {
	s.mu.RLock()
	registration, exists := s.services[req.Msg.TargetService]
	s.mu.RUnlock()

	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("target service not found: %s", req.Msg.TargetService))
	}

	log.WithFields(log.Fields{
		"target_service": req.Msg.TargetService,
		"operation":      req.Msg.Operation,
		"endpoint":       registration.Endpoint,
	}).Debug("Routing message")

	// TODO: Implement actual message routing to target service
	// For now, return a success response
	response := &brokerv1.RouteMessageResponse{
		Payload: []byte(fmt.Sprintf("Message routed to %s operation %s", req.Msg.TargetService, req.Msg.Operation)),
		Headers: map[string]string{
			"routed_at": time.Now().Format(time.RFC3339),
			"broker":    "plantd-broker",
		},
		Error: nil, // No error
	}

	s.lastActivity = time.Now()
	return connect.NewResponse(response), nil
}

// BroadcastMessage broadcasts a message to matching services
func (s *BrokerGRPCServer) BroadcastMessage(ctx context.Context, req *connect.Request[brokerv1.BroadcastMessageRequest]) (*connect.Response[emptypb.Empty], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pattern := req.Msg.ServicePattern
	matchCount := 0

	log.WithFields(log.Fields{
		"pattern":   pattern,
		"operation": req.Msg.Operation,
	}).Debug("Broadcasting message")

	for serviceID := range s.services {
		// Simple pattern matching (contains)
		if strings.Contains(serviceID, strings.ReplaceAll(pattern, "*", "")) {
			matchCount++
			// TODO: Implement actual broadcast to matching services
			log.WithFields(log.Fields{
				"service_id": serviceID,
				"operation":  req.Msg.Operation,
			}).Debug("Broadcasting to service")
		}
	}

	log.WithFields(log.Fields{
		"pattern":     pattern,
		"match_count": matchCount,
	}).Info("Broadcast completed")

	s.lastActivity = time.Now()
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// Heartbeat processes service heartbeats
func (s *BrokerGRPCServer) Heartbeat(ctx context.Context, req *connect.Request[brokerv1.HeartbeatRequest]) (*connect.Response[brokerv1.HeartbeatResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	registration, exists := s.services[req.Msg.ServiceId]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("service not registered: %s", req.Msg.ServiceId))
	}

	if registration.RegistrationID != req.Msg.RegistrationId {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("invalid registration ID"))
	}

	now := time.Now()
	registration.LastHeartbeat = now
	registration.Status = brokerv1.ServiceStatus_HEALTHY

	// Update metrics if provided
	if req.Msg.Metrics != nil {
		for key, value := range req.Msg.Metrics {
			registration.Metadata[fmt.Sprintf("metric_%s", key)] = value
		}
	}

	config := GetConfig()
	nextHeartbeat := now.Add(time.Duration(config.HeartbeatInterval) * time.Microsecond)

	response := &brokerv1.HeartbeatResponse{
		NextHeartbeat:       timestamppb.New(nextHeartbeat),
		HeartbeatIntervalMs: int32(config.HeartbeatInterval / 1000), // Convert to milliseconds
	}

	s.lastActivity = now
	return connect.NewResponse(response), nil
}

// GetServiceStatus returns the status of a specific service
func (s *BrokerGRPCServer) GetServiceStatus(ctx context.Context, req *connect.Request[brokerv1.GetServiceStatusRequest]) (*connect.Response[brokerv1.GetServiceStatusResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	registration, exists := s.services[req.Msg.ServiceId]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("service not found: %s", req.Msg.ServiceId))
	}

	// Check if service is healthy based on last heartbeat
	status := registration.Status
	message := "OK"

	config := GetConfig()
	heartbeatTimeout := time.Duration(config.HeartbeatInterval*config.HeartbeatLiveness) * time.Microsecond
	if time.Since(registration.LastHeartbeat) > heartbeatTimeout {
		status = brokerv1.ServiceStatus_TIMEOUT
		message = "Heartbeat timeout"
	}

	serviceStatus := &brokerv1.ServiceStatus{
		Status:    status,
		Message:   message,
		Metrics:   registration.Metadata,
		LastCheck: timestamppb.New(registration.LastHeartbeat),
	}

	response := &brokerv1.GetServiceStatusResponse{
		Status: serviceStatus,
	}

	return connect.NewResponse(response), nil
}

// CreateBus creates a new message bus
func (s *BrokerGRPCServer) CreateBus(ctx context.Context, req *connect.Request[brokerv1.CreateBusRequest]) (*connect.Response[brokerv1.CreateBusResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buses[req.Msg.Name]; exists {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("bus already exists: %s", req.Msg.Name))
	}

	now := time.Now()
	busID := fmt.Sprintf("bus-%s-%d", req.Msg.Name, now.Unix())

	bus := &BusRegistration{
		BusID:           busID,
		Name:            req.Msg.Name,
		Type:            req.Msg.Type,
		Config:          req.Msg.Config,
		SubscriberCount: 0,
		PublisherCount:  0,
		CreatedAt:       now,
	}

	s.buses[req.Msg.Name] = bus

	log.WithFields(log.Fields{
		"bus_name": req.Msg.Name,
		"bus_type": req.Msg.Type.String(),
		"bus_id":   busID,
	}).Info("Created message bus")

	response := &brokerv1.CreateBusResponse{
		BusId:     busID,
		CreatedAt: timestamppb.New(now),
	}

	s.lastActivity = now
	return connect.NewResponse(response), nil
}

// DeleteBus removes a message bus
func (s *BrokerGRPCServer) DeleteBus(ctx context.Context, req *connect.Request[brokerv1.DeleteBusRequest]) (*connect.Response[emptypb.Empty], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find bus by ID
	var busName string
	for name, bus := range s.buses {
		if bus.BusID == req.Msg.BusId {
			busName = name
			break
		}
	}

	if busName == "" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("bus not found: %s", req.Msg.BusId))
	}

	delete(s.buses, busName)

	log.WithFields(log.Fields{
		"bus_id":   req.Msg.BusId,
		"bus_name": busName,
	}).Info("Deleted message bus")

	s.lastActivity = time.Now()
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ListBuses streams all message buses
func (s *BrokerGRPCServer) ListBuses(ctx context.Context, req *connect.Request[brokerv1.ListBusesRequest], stream *connect.ServerStream[brokerv1.ListBusesResponse]) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	typeFilter := req.Msg.TypeFilter

	for _, bus := range s.buses {
		// Apply type filter if provided
		if typeFilter != brokerv1.BusType_BUS_TYPE_UNKNOWN && bus.Type != typeFilter {
			continue
		}

		busInfo := &brokerv1.BusInfo{
			BusId:           bus.BusID,
			Name:            bus.Name,
			Type:            bus.Type,
			Config:          bus.Config,
			SubscriberCount: bus.SubscriberCount,
			PublisherCount:  bus.PublisherCount,
			CreatedAt:       timestamppb.New(bus.CreatedAt),
		}

		response := &brokerv1.ListBusesResponse{
			Bus: busInfo,
		}

		if err := stream.Send(response); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send bus: %v", err))
		}
	}

	return nil
}

// PublishToBus publishes a message to a specific bus
func (s *BrokerGRPCServer) PublishToBus(ctx context.Context, req *connect.Request[brokerv1.PublishToBusRequest]) (*connect.Response[emptypb.Empty], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bus, exists := s.buses[req.Msg.BusName]
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("bus not found: %s", req.Msg.BusName))
	}

	log.WithFields(log.Fields{
		"bus_name": req.Msg.BusName,
		"topic":    req.Msg.Topic,
		"size":     len(req.Msg.Payload),
	}).Debug("Publishing to bus")

	// TODO: Implement actual bus publishing
	bus.PublisherCount++ // Simulate activity

	s.lastActivity = time.Now()
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// SubscribeToBus subscribes to messages from a specific bus
func (s *BrokerGRPCServer) SubscribeToBus(ctx context.Context, req *connect.Request[brokerv1.SubscribeToBusRequest], stream *connect.ServerStream[brokerv1.SubscribeToBusResponse]) error {
	s.mu.Lock()
	bus, exists := s.buses[req.Msg.BusName]
	if !exists {
		s.mu.Unlock()
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("bus not found: %s", req.Msg.BusName))
	}

	bus.SubscriberCount++
	s.mu.Unlock()

	log.WithFields(log.Fields{
		"bus_name": req.Msg.BusName,
		"topics":   req.Msg.Topics,
	}).Info("Subscribing to bus")

	// TODO: Implement actual bus subscription
	// For now, send a sample message
	response := &brokerv1.SubscribeToBusResponse{
		Topic:     "sample",
		Payload:   []byte("Sample message from bus"),
		Headers:   map[string]string{"bus": req.Msg.BusName},
		Timestamp: timestamppb.New(time.Now()),
	}

	if err := stream.Send(response); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to send message: %v", err))
	}

	s.lastActivity = time.Now()
	return nil
}

// Health returns the health status of the broker service
func (s *BrokerGRPCServer) Health(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[healthv1.HealthCheckResponse], error) {
	s.mu.RLock()
	serviceCount := len(s.services)
	busCount := len(s.buses)
	uptime := time.Since(s.startTime)
	lastActivity := s.lastActivity
	s.mu.RUnlock()

	status := healthv1.HealthCheckResponse_SERVING_STATUS_SERVING

	response := &healthv1.HealthCheckResponse{
		Status:    status,
		Message:   "Broker service is healthy",
		Timestamp: timestamppb.New(time.Now()),
		Uptime:    durationpb.New(uptime),
		Details: map[string]string{
			"service":        "broker",
			"version":        "1.0.0",
			"uptime_seconds": fmt.Sprintf("%.0f", uptime.Seconds()),
			"service_count":  fmt.Sprintf("%d", serviceCount),
			"bus_count":      fmt.Sprintf("%d", busCount),
			"last_activity":  lastActivity.Format(time.RFC3339),
		},
	}

	return connect.NewResponse(response), nil
}
