# MDP Implementation Upgrade Execution Plan

## Overview

This execution plan details the step-by-step process for upgrading the plantd MDP implementation from version 0.1 to 0.2, addressing critical issues, and enhancing reliability and performance.

## Prerequisites

- Go 1.19+ development environment
- ZeroMQ 4.x with Go bindings (goczmq)
- Comprehensive understanding of MDP v0.1 and v0.2 specifications
- Access to ZeroMQ reference implementations for testing

## Phase 1: Foundation & Critical Fixes (2-3 weeks)

### 1.1 Critical Issue Resolution (Week 1)

#### Task 1.1.1: Fix Frame Validation Issues
**Priority**: Critical  
**Estimated Duration**: 2-3 days

**Files to modify**:
- `core/mdp/client.go` (lines 162-177)
- `core/mdp/worker.go` (lines 191-210)
- `core/mdp/broker.go` (message handling sections)

**Implementation Steps**:

1. Create robust frame validation functions:
```go
// Add to util.go
func ValidateClientMessage(frames []string) error {
    if len(frames) < 4 {
        return fmt.Errorf("client message must have at least 4 frames, got %d", len(frames))
    }
    if frames[0] != "" {
        return fmt.Errorf("frame 0 must be empty for REQ emulation")
    }
    if frames[1] != MdpcClient {
        return fmt.Errorf("frame 1 must be %s, got %s", MdpcClient, frames[1])
    }
    return nil
}

func ValidateWorkerMessage(frames []string) error {
    if len(frames) < 3 {
        return fmt.Errorf("worker message must have at least 3 frames, got %d", len(frames))
    }
    if frames[0] != "" {
        return fmt.Errorf("frame 0 must be empty")
    }
    if frames[1] != MdpwWorker {
        return fmt.Errorf("frame 1 must be %s, got %s", MdpwWorker, frames[1])
    }
    return nil
}
```

2. Update client.go Recv() method:
```go
func (c *Client) Recv() (msg []string, err error) {
    // ... existing polling logic ...
    
    if len(recvMsg) > 0 {
        if err := ValidateClientMessage(recvMsg); err != nil {
            log.WithError(err).Error("received invalid client message")
            return nil, fmt.Errorf("invalid message format: %w", err)
        }
        
        service := recvMsg[2]
        msg = recvMsg[3:]
        
        log.WithFields(log.Fields{"service": service, "msg": msg}).Debug("received message")
        return msg, nil
    }
    
    // ... rest of method ...
}
```

3. Update worker.go Recv() method with proper validation
4. Add comprehensive error handling for malformed messages

#### Task 1.1.2: Implement Proper Error Handling
**Priority**: High  
**Estimated Duration**: 3-4 days

**Files to modify**:
- `core/mdp/errors.go` (expand error definitions)
- All major files (add error wrapping and context)

**Implementation Steps**:

1. Expand error definitions:
```go
// errors.go
var (
    ErrInvalidMessage    = errors.New("invalid message format")
    ErrProtocolViolation = errors.New("protocol violation")
    ErrTimeout          = errors.New("operation timeout")
    ErrBrokerUnavailable = errors.New("broker unavailable")
    ErrServiceNotFound  = errors.New("service not found")
    ErrWorkerDisconnected = errors.New("worker disconnected")
)

type MDPError struct {
    Code    string
    Message string
    Cause   error
}

func (e *MDPError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("MDP %s: %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("MDP %s: %s", e.Code, e.Message)
}

func (e *MDPError) Unwrap() error {
    return e.Cause
}
```

2. Implement circuit breaker pattern for broker connections
3. Add exponential backoff for reconnection attempts
4. Implement graceful degradation strategies

#### Task 1.1.3: Add Comprehensive Test Coverage
**Priority**: High  
**Estimated Duration**: 4-5 days

**New files to create**:
- `core/mdp/integration_test.go`
- `core/mdp/protocol_test.go`
- `core/mdp/reliability_test.go`

**Implementation Steps**:

1. Create protocol compliance tests:
```go
// protocol_test.go
func TestMDPv1ProtocolCompliance(t *testing.T) {
    // Test all message formats according to spec
    testCases := []struct {
        name     string
        message  []string
        expected error
    }{
        // Add test cases for all message types
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Validate message format
        })
    }
}
```

2. Add integration tests with real broker/client/worker interactions
3. Implement failure injection tests
4. Add performance benchmarks

### 1.2 MMI Implementation (Week 2)

#### Task 1.2.1: Basic MMI Framework
**Priority**: High  
**Estimated Duration**: 3-4 days

**Files to create/modify**:
- `core/mdp/mmi.go` (new file)
- `core/mdp/broker.go` (add MMI handling)
- `core/mdp/const.go` (add MMI constants)

**Implementation Steps**:

1. Create MMI constants:
```go
// const.go - Add MMI service definitions
const (
    MMINamespace = "mmi."
    
    // Standard MMI services
    MMIService   = "mmi.service"
    MMIWorkers   = "mmi.workers"  
    MMIHeartbeat = "mmi.heartbeat"
    MMIBroker    = "mmi.broker"
)

// MMI response codes
const (
    MMICodeOK           = "200"
    MMICodeNotFound     = "404"
    MMICodeNotImplemented = "501"
    MMICodeError        = "500"
)
```

2. Implement MMI service handler:
```go
// mmi.go
type MMIHandler struct {
    broker *Broker
}

func NewMMIHandler(broker *Broker) *MMIHandler {
    return &MMIHandler{broker: broker}
}

func (m *MMIHandler) HandleRequest(service string, request []string) ([]string, error) {
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
        return []string{MMICodeNotImplemented}, nil
    }
}

func (m *MMIHandler) handleServiceQuery(request []string) ([]string, error) {
    if len(request) < 1 {
        return []string{MMICodeError, "service name required"}, nil
    }
    
    serviceName := request[0]
    service, exists := m.broker.services[serviceName]
    
    if !exists || len(service.waiting) == 0 {
        return []string{MMICodeNotFound}, nil
    }
    
    return []string{MMICodeOK}, nil
}
```

3. Integrate MMI handler into broker

#### Task 1.2.2: Extended MMI Services
**Priority**: Medium  
**Estimated Duration**: 2-3 days

**Implementation Steps**:

1. Implement broker information service
2. Add worker statistics service
3. Create service discovery endpoints
4. Add health check endpoints

### 1.3 Configuration System (Week 2-3)

#### Task 1.3.1: Configurable Parameters
**Priority**: Medium  
**Estimated Duration**: 2-3 days

**Files to modify**:
- `core/mdp/config.go` (new file)
- `core/mdp/const.go` (make configurable)
- All component files (use config)

**Implementation Steps**:

1. Create configuration structure:
```go
// config.go
type Config struct {
    HeartbeatInterval    time.Duration `yaml:"heartbeat_interval" default:"2500ms"`
    HeartbeatLiveness    int           `yaml:"heartbeat_liveness" default:"3"`
    ReconnectInterval    time.Duration `yaml:"reconnect_interval" default:"2500ms"`
    RequestTimeout       time.Duration `yaml:"request_timeout" default:"5000ms"`
    MaxRetries          int           `yaml:"max_retries" default:"3"`
    SocketHWM           int           `yaml:"socket_hwm" default:"1000"`
    LogLevel            string        `yaml:"log_level" default:"info"`
    EnableMetrics       bool          `yaml:"enable_metrics" default:"true"`
    EnableMMI          bool          `yaml:"enable_mmi" default:"true"`
}

func LoadConfig(filename string) (*Config, error) {
    // Load from YAML file with defaults
}

func (c *Config) Validate() error {
    // Validate configuration parameters
}
```

2. Update all components to use configuration
3. Add environment variable overrides
4. Implement configuration validation

## Phase 2: MDP v0.2 Protocol Upgrade (2-3 weeks)

### 2.1 Protocol Version Update (Week 4)

#### Task 2.1.1: Update Protocol Identifiers
**Priority**: High  
**Estimated Duration**: 1-2 days

**Files to modify**:
- `core/mdp/const.go`
- All files using protocol constants

**Implementation Steps**:

1. Update protocol constants:
```go
// const.go
const (
    // MDP v0.2 protocol identifiers
    MdpcClient = "MDPC02"
    MdpwWorker = "MDPW02"
    
    // Backward compatibility (if needed)
    MdpcClientV1 = "MDPC01"
    MdpwWorkerV1 = "MDPW01"
)
```

2. Add version detection logic
3. Implement backward compatibility layer (optional)

#### Task 2.1.2: Remove Empty Frame Handling
**Priority**: High  
**Estimated Duration**: 2-3 days

**Files to modify**:
- `core/mdp/client.go`
- `core/mdp/worker.go`
- `core/mdp/broker.go`

**Implementation Steps**:

1. Update client message format (remove empty frame):
```go
// client.go - Updated Send method
func (c *Client) Send(service string, request ...string) (err error) {
    // MDP v0.2 format - no empty frame
    req := make([]string, 3, len(request)+3)
    req = append(req, request...)
    req[2] = service
    req[1] = MdpcClient
    req[0] = string(rune(0x01)) // REQUEST command
    
    err = c.client.SendMessage(stringArrayToByte2D(req))
    return
}
```

2. Update worker message format
3. Update broker message parsing
4. Remove all empty frame validation

### 2.2 PARTIAL/FINAL Commands (Week 4-5)

#### Task 2.2.1: Implement New Command Types
**Priority**: High  
**Estimated Duration**: 3-4 days

**Files to modify**:
- `core/mdp/const.go` (add new commands)
- `core/mdp/worker.go` (implement PARTIAL/FINAL)
- `core/mdp/client.go` (handle PARTIAL/FINAL)
- `core/mdp/broker.go` (route PARTIAL/FINAL)

**Implementation Steps**:

1. Add new command constants:
```go
// const.go - MDP v0.2 commands
const (
    // Client commands
    MdpcRequest = string(rune(0x01))
    
    // Client replies
    MdpcPartial = string(rune(0x02))
    MdpcFinal   = string(rune(0x03))
    
    // Worker commands  
    MdpwReady      = string(rune(0x01))
    MdpwRequest    = string(rune(0x02))
    MdpwPartial    = string(rune(0x03))
    MdpwFinal      = string(rune(0x04))
    MdpwHeartbeat  = string(rune(0x05))
    MdpwDisconnect = string(rune(0x06))
)
```

2. Implement streaming response API:
```go
// worker.go - Add streaming response methods
type ResponseStream struct {
    worker *Worker
    client string
}

func (r *ResponseStream) SendPartial(data []string) error {
    return r.worker.sendReply(MdpwPartial, r.client, data)
}

func (r *ResponseStream) SendFinal(data []string) error {
    return r.worker.sendReply(MdpwFinal, r.client, data)
}

func (w *Worker) RecvStream() (*ResponseStream, []string, error) {
    // Modified Recv that returns a stream handler
}
```

3. Update client to handle multiple responses:
```go
// client.go - Add streaming receive
func (c *Client) RecvStream() (chan []string, error) {
    responseChan := make(chan []string, 10)
    
    go func() {
        defer close(responseChan)
        for {
            msg, err := c.recvSingle()
            if err != nil {
                return
            }
            
            command := msg[1]
            data := msg[3:]
            
            responseChan <- data
            
            if command == MdpcFinal {
                return
            }
        }
    }()
    
    return responseChan, nil
}
```

#### Task 2.2.2: Update API Interfaces
**Priority**: High  
**Estimated Duration**: 2-3 days

**Implementation Steps**:

1. Design backward-compatible APIs
2. Update client API for streaming
3. Update worker API for streaming
4. Add examples and documentation

### 2.3 Socket Requirements Update (Week 5)

#### Task 2.3.1: Enforce DEALER Socket Usage
**Priority**: Medium  
**Estimated Duration**: 1-2 days

**Files to modify**:
- `core/mdp/client.go`
- Documentation and examples

**Implementation Steps**:

1. Remove REQ socket references
2. Update documentation
3. Add socket type validation
4. Update examples

## Phase 3: Reliability & Performance (3-4 weeks)

### 3.1 Request Durability (Week 6)

#### Task 3.1.1: Implement Request Persistence
**Priority**: High  
**Estimated Duration**: 4-5 days

**Files to create/modify**:
- `core/mdp/persistence.go` (new file)
- `core/mdp/broker.go` (integrate persistence)

**Implementation Steps**:

1. Create persistence interface:
```go
// persistence.go
type PersistenceStore interface {
    StoreRequest(id string, request *Request) error
    RetrieveRequest(id string) (*Request, error)
    DeleteRequest(id string) error
    ListPendingRequests() ([]string, error)
}

type Request struct {
    ID        string
    Client    string
    Service   string
    Data      []string
    Timestamp time.Time
    Retries   int
}
```

2. Implement in-memory and persistent stores
3. Add request replay logic
4. Implement request deduplication

#### Task 3.1.2: Transaction Logging
**Priority**: Medium  
**Estimated Duration**: 3-4 days

**Implementation Steps**:

1. Design transaction log format
2. Implement log rotation
3. Add recovery procedures
4. Create log analysis tools

### 3.2 Broker Clustering (Week 7)

#### Task 3.2.1: Broker Discovery
**Priority**: High  
**Estimated Duration**: 5-6 days

**Implementation Steps**:

1. Implement broker discovery protocol
2. Add cluster membership management
3. Create load balancing strategies
4. Implement failover detection

#### Task 3.2.2: State Synchronization
**Priority**: High  
**Estimated Duration**: 4-5 days

**Implementation Steps**:

1. Design state synchronization protocol
2. Implement worker state replication
3. Add service registry synchronization
4. Create conflict resolution mechanisms

### 3.3 Performance Optimizations (Week 8-9)

#### Task 3.3.1: Message Pooling
**Priority**: Medium  
**Estimated Duration**: 3-4 days

**Implementation Steps**:

1. Implement message object pooling
2. Add buffer reuse mechanisms
3. Optimize memory allocations
4. Profile and benchmark improvements

#### Task 3.3.2: Asynchronous I/O
**Priority**: Medium  
**Estimated Duration**: 4-5 days

**Implementation Steps**:

1. Implement async message handling
2. Add concurrent request processing
3. Optimize critical path performance
4. Add performance monitoring

## Phase 4: Security Enhancement (2-3 weeks)

### 4.1 Authentication Framework (Week 10)

#### Task 4.1.1: ZAP Integration
**Priority**: High  
**Estimated Duration**: 5-6 days

**Files to create**:
- `core/mdp/auth.go`
- `core/mdp/zap.go`

**Implementation Steps**:

1. Implement ZAP authentication handler
2. Add credential management
3. Create authentication policies
4. Integrate with existing components

### 4.2 Encryption Support (Week 11)

#### Task 4.2.1: CURVE Implementation
**Priority**: High  
**Estimated Duration**: 4-5 days

**Implementation Steps**:

1. Add CURVE encryption support
2. Implement key management
3. Create secure connection establishment
4. Add certificate handling

### 4.3 Access Control (Week 11-12)

#### Task 4.3.1: Service Authorization
**Priority**: Medium  
**Estimated Duration**: 3-4 days

**Implementation Steps**:

1. Design authorization framework
2. Implement service access controls
3. Add role-based permissions
4. Create policy management interface

## Testing & Validation

### Integration Testing
- Test with official ZeroMQ reference implementations
- Verify interoperability with other MDP implementations
- Performance testing under various load conditions
- Failure scenario testing

### Compliance Testing
- Protocol compliance verification
- Message format validation
- Behavioral testing against specifications
- Security vulnerability assessment

## Risk Mitigation

### Technical Risks
1. **Protocol Incompatibility**: Maintain backward compatibility layer
2. **Performance Regression**: Continuous benchmarking
3. **Data Loss**: Comprehensive testing of persistence layer
4. **Security Vulnerabilities**: Security audits and penetration testing

### Implementation Risks
1. **Timeline Overrun**: Prioritize critical features first
2. **Resource Constraints**: Modular implementation allows partial deployment
3. **Integration Issues**: Extensive integration testing

## Success Criteria

### Phase 1 Success Criteria
- [ ] All critical frame validation issues resolved
- [ ] Basic MMI services operational
- [ ] Comprehensive test coverage > 80%
- [ ] Zero protocol compliance failures

### Phase 2 Success Criteria
- [ ] Full MDP v0.2 protocol compliance
- [ ] PARTIAL/FINAL command support
- [ ] Streaming response functionality
- [ ] Backward compatibility maintained

### Phase 3 Success Criteria
- [ ] Request durability implemented
- [ ] Broker clustering operational
- [ ] Performance improvements demonstrated
- [ ] High availability achieved

### Phase 4 Success Criteria
- [ ] Authentication framework operational
- [ ] Encryption support available
- [ ] Access control system functional
- [ ] Security audit passed

## Rollback Plan

### Emergency Rollback
1. Maintain feature flags for new functionality
2. Keep v0.1 implementation as fallback
3. Database/state migration rollback procedures
4. Configuration rollback mechanisms

### Gradual Rollback
1. Component-by-component rollback capability
2. A/B testing infrastructure
3. Monitoring and alerting for issues
4. Automated rollback triggers

## Dependencies

### External Dependencies
- ZeroMQ 4.x library updates
- Go module dependencies
- Testing framework tools
- Security audit tools

### Internal Dependencies
- Configuration management system
- Logging infrastructure
- Monitoring and metrics
- Documentation updates

## Deliverables

### Documentation
- [ ] Updated API documentation
- [ ] Migration guide from v0.1 to v0.2
- [ ] Configuration reference
- [ ] Security implementation guide
- [ ] Performance tuning guide

### Code Deliverables
- [ ] Upgraded MDP v0.2 implementation
- [ ] Comprehensive test suite
- [ ] Performance benchmarks
- [ ] Security assessment report
- [ ] Migration tools and scripts

This execution plan provides a structured approach to upgrading the MDP implementation while maintaining system stability and adding robust new features for production deployment. 
