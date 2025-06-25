# SSE Implementation and ZeroMQ CGO Signal Stack Corruption - Root Cause Analysis

## Executive Summary

This document provides a comprehensive root cause analysis of Server-Sent Events (SSE) implementation issues that led to the discovery of underlying ZeroMQ CGO signal stack corruption in the plantd App Service. While the initial problem appeared to be SSE-related, investigation revealed critical resource management issues in the ZeroMQ connection handling that caused service crashes.

**Date**: 2024-12-19
**Affected Service**: App Service
**Severity**: High
**Status**: Partially Resolved (SSE working, underlying ZeroMQ issue persists)

## 🔍 Root Cause Analysis: ZeroMQ CGO Signal Stack Corruption

### Initial Problem Statement

The App Service dashboard was not receiving real-time updates via Server-Sent Events (SSE). Users reported that dashboard metrics (requests per second, uptime, service health) were not updating in real-time, with no data being received from the SSE endpoint.

### Investigation Timeline

#### Phase 1: SSE Implementation Issues
**Initial Findings:**
- Improper SSE streaming implementation not using `fasthttp.StreamWriter`
- Missing keep-alive mechanism for maintaining connections
- JavaScript error handling insufficient for null/undefined data
- Missing script tag in dashboard template for loading `dashboard.js`

**Initial Fixes Applied:**
- Modified SSE handler to use proper streaming with buffered writing and flushing
- Added 30-second keep-alive intervals
- Enhanced JavaScript error handling and debugging
- Added script tag to dashboard template
- Rebuilt and restarted service

#### Phase 2: Service Crash Discovery
**Critical Issue Identified:**
After implementing SSE fixes, service began crashing after ~30 seconds with:

```
Fasthttp worker pool crashes
Dangling ZeroMQ DEALER socket errors
Stack traces pointing to health_service.go:126
```

**Error Pattern:**
```
fatal error: unexpected signal during runtime execution
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x0]

runtime stack:
runtime.throw(0x1a4b9a0, 0x2a)
	/usr/local/go/src/runtime/panic.go:1116 +0x72
runtime.sigpanic()
	/usr/local/go/src/runtime/signal_unix.go:718 +0x4ac

goroutine 47 [running]:
app/internal/services.(*HealthService).GetSystemHealth.func2(0xc00015e780, 0xc000186240)
	/Users/geoff/Projects/plantd/app/internal/services/health_service.go:126 +0x88
created by app/internal/services.(*HealthService).GetSystemHealth
	/Users/geoff/Projects/plantd/app/internal/services/health_service.go:120 +0x1c5
```

#### Phase 3: Resource Management Investigation
**Root Cause Identified:**
- Resource leaks in `fasthttp.StreamWriter` causing memory corruption
- ZeroMQ DEALER sockets created for health checks not being properly cleaned up
- Concurrent goroutines in health service spawning multiple socket connections
- MDP (Majordomo Protocol) client implementation with socket leaks

### Technical Root Cause Analysis

#### 1. ZeroMQ Socket Lifecycle Issues

**Problem Location**: `health_service.go:126`
```go
// Line 126 in GetSystemHealth goroutine checking broker health
err := hs.brokerService.CheckConnectivity()
```

**Underlying Issue**: The `CheckConnectivity` method creates ZeroMQ DEALER sockets through the MDP client but fails to properly clean them up, especially under concurrent access patterns.

#### 2. CGO Signal Stack Corruption

**Technical Details:**
- ZeroMQ uses CGO to interface with native C libraries
- Improperly cleaned up sockets leave dangling file descriptors
- Memory corruption occurs when CGO stack becomes inconsistent
- Signal handlers (SIGSEGV) trigger during garbage collection attempts on corrupted memory

#### 3. Concurrency Amplification

**Problem Pattern:**
```go
// Multiple goroutines creating concurrent socket connections
go func() {
    err := hs.brokerService.CheckConnectivity() // Creates ZMQ DEALER socket
    // Socket cleanup not guaranteed on error/panic
}()
```

**Amplification Factor:**
- SSE connections trigger frequent health checks
- Each health check spawns multiple goroutines
- Each goroutine potentially creates ZeroMQ sockets
- Failed cleanup multiplies resource leaks

### Attempted Solutions and Results

#### Solution 1: FastHTTP StreamWriter Resource Management
**Approach:**
- Added active SSE connection tracking with unique stream IDs
- Implemented graceful shutdown with `CleanupActiveStreams()`
- Added circuit breaker protection (5 failure threshold)
- Added timeout protection (1s for dashboard, 500ms for status updates)

**Result:** Service still crashed with same ZeroMQ socket errors

#### Solution 2: Circuit Breaker Protection
**Approach:**
- Implemented comprehensive circuit breaker with atomic operations
- Added failure tracking and 30-second recovery
- Added graceful degradation when services unavailable

**Result:** Reduced crash frequency but core issue persisted

#### Solution 3: Simplified SSE Implementation
**Approach:**
- Completely removed `fasthttp.StreamWriter` approach
- Implemented direct `fiber.Ctx.WriteString()` for SSE
- Disabled problematic StreamWriter methods

**Result:** SSE functionality restored, but underlying ZeroMQ crashes continued

### Current Status and Impact

#### ✅ **Resolved Issues:**
1. **SSE Functionality**: Server-Sent Events now working correctly
2. **Dashboard Updates**: Real-time metrics displaying properly
3. **Connection Management**: SSE connections stable and reliable
4. **Circuit Breaker**: Service degradation handling implemented

#### ⚠️  **Persistent Issues:**
1. **ZeroMQ Socket Leaks**: Underlying socket cleanup issues remain
2. **Service Crashes**: Intermittent crashes still occurring
3. **CGO Stack Corruption**: Memory corruption in ZeroMQ CGO interface
4. **Health Check Instability**: Concurrent health checks causing resource exhaustion

### Recommended Actions

#### Immediate (High Priority)
1. **ZeroMQ Connection Pooling**: Implement connection pooling in MDP client to reuse sockets
2. **Health Check Serialization**: Serialize health checks to prevent concurrent socket creation
3. **Resource Monitoring**: Add ZeroMQ socket monitoring and cleanup verification
4. **Graceful Error Handling**: Improve error handling in MDP client socket operations

#### Medium Term (Medium Priority)
1. **MDP Client Refactoring**: Review and refactor MDP client implementation for proper resource management
2. **Alternative Health Check**: Consider HTTP-based health checks as fallback
3. **Socket Lifecycle Management**: Implement explicit socket lifecycle management with defer cleanup
4. **Memory Profiling**: Add memory profiling to detect resource leaks early

#### Long Term (Low Priority)
1. **ZeroMQ Alternatives**: Evaluate alternative messaging protocols for health checks
2. **Service Architecture**: Consider microservice patterns to isolate ZeroMQ usage
3. **Resource Limits**: Implement resource limits and monitoring for socket usage
4. **CGO Safety**: Review CGO usage patterns for memory safety

### Lessons Learned

#### Technical Insights
1. **Resource Management**: CGO libraries require explicit resource management
2. **Concurrency Patterns**: Concurrent socket creation amplifies resource leaks
3. **Error Propagation**: Socket errors can cause memory corruption in CGO context
4. **Debugging Complexity**: CGO signal stack corruption difficult to debug

#### Process Improvements
1. **Resource Testing**: Need comprehensive resource leak testing
2. **CGO Validation**: Require explicit CGO resource management patterns
3. **Concurrency Review**: Review all concurrent access to external libraries
4. **Error Handling**: Standardize error handling for external library interfaces

### Appendix: Error Signatures

#### ZeroMQ Socket Leak Signature
```
Dangling ZeroMQ DEALER socket errors
Context termination timeout
Socket close failure
```

#### CGO Signal Stack Corruption Signature
```
fatal error: unexpected signal during runtime execution
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x0]
runtime.sigpanic()
```

#### Health Service Crash Pattern
```
goroutine [running]:
app/internal/services.(*HealthService).GetSystemHealth.func2
health_service.go:126
```

### References

- [ZeroMQ Socket Management Best Practices](https://zeromq.org/socket-api/)
- [Go CGO Memory Management](https://golang.org/cmd/cgo/)
- [Majordomo Protocol Specification](https://rfc.zeromq.org/spec/7/)
- [FastHTTP StreamWriter Documentation](https://pkg.go.dev/github.com/valyala/fasthttp)

---

**Document Version**: 1.0  
**Last Updated**: 2024-12-19  
**Next Review**: 2024-12-26  
**Owner**: plantd Development Team 
