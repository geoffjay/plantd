# Communication Patterns Analysis

## Overview

Plantd implements a sophisticated messaging architecture based on ZeroMQ with custom protocol implementations. The system uses multiple communication patterns to achieve different reliability, performance, and scalability characteristics.

## Core Communication Patterns

### 1. Request-Reply Pattern (Majordomo Protocol v2)

**Purpose**: Reliable, synchronous service communication with load balancing

**Implementation**: Custom MDP/2 protocol over ZeroMQ REQ/REP sockets

**Message Flow**:
```
Client → Broker → Worker → Broker → Client
   │        │        │        │        │
   │        │        │        │        │
REQUEST  FORWARD   PROCESS   REPLY   RESPONSE
```

**Key Features**:
- **Reliability**: Message persistence and delivery guarantees
- **Load Balancing**: Automatic distribution across available workers
- **Service Discovery**: Dynamic worker registration and health monitoring
- **Timeout Handling**: Configurable request timeouts with retry logic
- **Heartbeat Monitoring**: Worker health detection and automatic cleanup

**Protocol Messages**:
```
READY      - Worker announces availability
REQUEST    - Client request forwarded to worker
REPLY      - Worker response sent back to client
HEARTBEAT  - Keep-alive messages between broker and worker
DISCONNECT - Graceful worker shutdown notification
```

**Usage Examples**:
- State service operations (get, set, delete)
- Service health checks and status queries
- Command execution and control operations
- Configuration updates and management

### 2. Publish-Subscribe Pattern

**Purpose**: One-to-many message distribution for events and data streaming

**Implementation**: ZeroMQ PUB/SUB sockets with XPUB/XSUB proxy

**Architecture**:
```
Publishers → Frontend (XSUB) → Backend (XPUB) → Subscribers
                    ↓
               Capture Socket (optional)
```

**Message Flow**:
```
Publisher → Message Bus → Multiple Subscribers
    │           │              │
    │           │              │
 PUBLISH    DISTRIBUTE      RECEIVE
```

**Key Features**:
- **Scalability**: Efficient one-to-many distribution
- **Decoupling**: Publishers and subscribers operate independently
- **Topic Filtering**: Subscription-based message filtering
- **Message Capture**: Optional logging and debugging support
- **Dynamic Subscriptions**: Runtime subscription management

**Usage Examples**:
- State change notifications
- Metric data distribution
- Event logging and auditing
- Real-time monitoring updates
- Configuration change broadcasts

### 3. Pipeline Pattern

**Purpose**: Sequential data processing with flow control

**Implementation**: ZeroMQ PUSH/PULL sockets for work distribution

**Message Flow**:
```
Source → Stage 1 → Stage 2 → ... → Stage N → Sink
   │        │         │              │        │
   │        │         │              │        │
PRODUCE  PROCESS   PROCESS        PROCESS   CONSUME
```

**Key Features**:
- **Flow Control**: Backpressure handling and buffering
- **Parallel Processing**: Multiple workers per stage
- **Fault Tolerance**: Failed message handling and retry
- **Composability**: Modular processing stages

**Usage Examples**:
- Data ingestion and ETL pipelines
- Message processing workflows
- Batch job processing
- Stream processing applications

## Protocol Implementation Details

### Majordomo Protocol v2 (MDP/2)

**Protocol Specification**: Based on RFC 7 (http://rfc.zeromq.org/spec:7)

**Frame Structure**:
```
Frame 0: Empty frame (delimiter)
Frame 1: Protocol signature ("MDPC01" or "MDPW01")
Frame 2: Command (READY, REQUEST, REPLY, etc.)
Frame 3+: Command-specific data frames
```

**Client Protocol (MDPC01)**:
```go
// Request message format
[empty][MDPC01][service_name][request_data...]

// Reply message format  
[empty][MDPC01][service_name][reply_data...]
```

**Worker Protocol (MDPW01)**:
```go
// READY message
[empty][MDPW01][READY][service_name]

// REQUEST message (from broker to worker)
[empty][MDPW01][REQUEST][client_id][empty][request_data...]

// REPLY message (from worker to broker)
[empty][MDPW01][REPLY][client_id][empty][reply_data...]

// HEARTBEAT message
[empty][MDPW01][HEARTBEAT]
```

**Heartbeat Mechanism**:
- **Interval**: 2.5 seconds between heartbeats
- **Liveness**: 3 missed heartbeats before worker considered dead
- **Expiry**: 7.5 seconds total timeout for worker failure detection

### Message Bus Protocol

**Bus Configuration**:
```yaml
buses:
  - name: "metric"
    frontend: "tcp://*:11000"  # Publishers connect here
    backend: "tcp://*:11001"   # Subscribers connect here
    capture: "tcp://*:11002"   # Optional message capture
```

**Message Format**:
```
Topic Frame: Service namespace (e.g., "org.plantd.Metrics")
Data Frame:  JSON-encoded message payload
```

**Subscription Patterns**:
```go
// Subscribe to all messages from a service
subscriber.SetSubscribe("org.plantd.Service")

// Subscribe to specific message types
subscriber.SetSubscribe("org.plantd.Service.events")
```

## Service Communication Matrix

| Source Service | Target Service | Pattern | Protocol | Purpose |
|---------------|---------------|---------|----------|---------|
| Client | Broker | Request-Reply | MDP/2 | Service requests |
| Broker | State Worker | Request-Reply | MDP/2 | State operations |
| Broker | Logger Worker | Request-Reply | MDP/2 | Log aggregation |
| State Service | Message Bus | Pub-Sub | ZMQ PUB/SUB | State change events |
| Modules | Message Bus | Pub-Sub | ZMQ PUB/SUB | Data streaming |
| Proxy | All Services | Request-Reply | MDP/2 | Protocol translation |
| App | All Services | Request-Reply | MDP/2 | Web interface |

## Message Serialization

### JSON Format (Primary)
```json
{
  "service": "org.plantd.State",
  "operation": "set",
  "key": "configuration.timeout",
  "value": "30s",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Advantages**:
- Human-readable for debugging
- Wide language support
- Schema flexibility
- Web-friendly format

**Disadvantages**:
- Larger message size
- Parsing overhead
- No schema validation

### Binary Formats (Future)
- **MessagePack**: Compact binary JSON alternative
- **Protocol Buffers**: Schema-based serialization
- **Apache Avro**: Schema evolution support

## Error Handling and Reliability

### Error Categories

#### 1. Network Errors
- Connection failures and timeouts
- Socket errors and disconnections
- Network partitions and latency spikes

**Handling Strategy**:
```go
// Automatic reconnection with exponential backoff
func (c *Client) reconnect() error {
    backoff := time.Second
    for attempts := 0; attempts < maxRetries; attempts++ {
        if err := c.connect(); err == nil {
            return nil
        }
        time.Sleep(backoff)
        backoff *= 2
    }
    return errors.New("max reconnection attempts exceeded")
}
```

#### 2. Protocol Errors
- Invalid message formats
- Unknown commands or services
- Protocol version mismatches

**Handling Strategy**:
```go
// Protocol validation and error responses
func (b *Broker) validateMessage(frames []string) error {
    if len(frames) < 3 {
        return errors.New("insufficient message frames")
    }
    if frames[1] != MdpwWorker && frames[1] != MdpcClient {
        return errors.New("invalid protocol version")
    }
    return nil
}
```

#### 3. Application Errors
- Business logic failures
- Data validation errors
- Resource constraints

**Handling Strategy**:
```go
// Structured error responses
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

### Reliability Mechanisms

#### 1. Message Persistence
- **Broker Queue**: In-memory message queuing with optional persistence
- **State Persistence**: SQLite database for durable state storage
- **Dead Letter Queue**: Failed message preservation for analysis

#### 2. Delivery Guarantees
- **At-Least-Once**: Messages delivered at least once (may duplicate)
- **Exactly-Once**: Planned with message deduplication
- **Best-Effort**: Fire-and-forget for non-critical messages

#### 3. Circuit Breaker Pattern
```go
type CircuitBreaker struct {
    maxFailures int
    timeout     time.Duration
    failures    int
    lastFailure time.Time
    state       State // CLOSED, OPEN, HALF_OPEN
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == OPEN {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = HALF_OPEN
        } else {
            return errors.New("circuit breaker is open")
        }
    }
    
    err := fn()
    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        if cb.failures >= cb.maxFailures {
            cb.state = OPEN
        }
    } else {
        cb.failures = 0
        cb.state = CLOSED
    }
    
    return err
}
```

## Performance Characteristics

### Latency Measurements
- **Local IPC**: <100 microseconds
- **TCP Localhost**: <1 millisecond
- **Network TCP**: 1-10 milliseconds (depending on network)
- **Request-Reply**: 2x network latency + processing time

### Throughput Benchmarks
- **Broker Routing**: 100K+ messages/second
- **Pub-Sub Distribution**: 1M+ messages/second
- **State Operations**: 10K+ operations/second
- **Pipeline Processing**: Depends on processing complexity

### Memory Usage
- **Message Overhead**: ~100 bytes per message
- **Connection Overhead**: ~1KB per connection
- **Queue Memory**: Configurable with high-water marks
- **Total Memory**: Scales linearly with active connections

### CPU Usage
- **Message Routing**: <1% CPU for typical loads
- **JSON Serialization**: 5-10% CPU under heavy load
- **Network I/O**: Minimal with ZeroMQ's efficient implementation
- **Heartbeat Processing**: Negligible background load

## Scalability Patterns

### Horizontal Scaling

#### 1. Worker Scaling
```
Client → Broker → Worker Pool
                    ├── Worker 1
                    ├── Worker 2
                    └── Worker N
```

#### 2. Broker Federation
```
Client Pool → Broker Pool → Worker Pool
├── Client 1    ├── Broker 1    ├── Worker 1
├── Client 2    ├── Broker 2    ├── Worker 2
└── Client N    └── Broker N    └── Worker N
```

#### 3. Message Bus Clustering
```
Publishers → Bus Cluster → Subscribers
              ├── Bus 1
              ├── Bus 2
              └── Bus N
```

### Vertical Scaling
- **CPU Scaling**: Multi-threaded message processing
- **Memory Scaling**: Larger message queues and buffers
- **Network Scaling**: Higher bandwidth and connection limits
- **Storage Scaling**: Faster disk I/O for persistence

### Geographic Distribution
- **Regional Brokers**: Reduced latency for local clients
- **Data Replication**: State synchronization across regions
- **Failover Mechanisms**: Automatic regional failover
- **Load Balancing**: Geographic request routing

## Security Considerations

### Current Security Posture
- **Transport Security**: Plain TCP (no encryption)
- **Authentication**: Not implemented
- **Authorization**: No access controls
- **Message Integrity**: No signing or verification

### Security Vulnerabilities
1. **Eavesdropping**: Unencrypted message content
2. **Man-in-the-Middle**: No certificate validation
3. **Replay Attacks**: No message sequence numbers
4. **DoS Attacks**: No rate limiting or connection limits
5. **Injection Attacks**: Limited input validation

### Recommended Security Enhancements

#### 1. Transport Layer Security
```go
// TLS configuration for ZeroMQ
config := &tls.Config{
    Certificates: []tls.Certificate{cert},
    ClientAuth:   tls.RequireAndVerifyClientCert,
    ClientCAs:    caCertPool,
}
```

#### 2. Message Authentication
```go
// HMAC message signing
func signMessage(message []byte, key []byte) []byte {
    h := hmac.New(sha256.New, key)
    h.Write(message)
    return h.Sum(nil)
}
```

#### 3. Access Control
```go
// Role-based access control
type AccessControl struct {
    roles map[string][]string // user -> roles
    perms map[string][]string // role -> permissions
}

func (ac *AccessControl) CheckPermission(user, action string) bool {
    for _, role := range ac.roles[user] {
        for _, perm := range ac.perms[role] {
            if perm == action {
                return true
            }
        }
    }
    return false
}
```

## Future Communication Enhancements

### Planned Improvements
1. **gRPC Integration**: High-performance RPC with HTTP/2
2. **WebSocket Support**: Real-time web client communication
3. **Message Compression**: Reduced bandwidth usage
4. **Schema Registry**: Message format versioning and validation
5. **Distributed Tracing**: Request correlation across services
6. **Metrics Collection**: Built-in performance monitoring
7. **Rate Limiting**: DoS protection and fair usage
8. **Message Encryption**: End-to-end message security