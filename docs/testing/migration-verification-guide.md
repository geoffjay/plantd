# PlantD gRPC Migration Verification Guide

This guide provides comprehensive testing procedures to verify that the gRPC migration is complete and functionally equivalent to MDP before removing legacy dependencies.

## Overview

The PlantD system now supports **dual protocols** (MDP + gRPC) allowing for safe testing and migration verification. This guide ensures that:

1. ✅ All CLI commands work with both protocols
2. ✅ gRPC functionality is identical to MDP functionality  
3. ✅ Services can operate in dual-protocol mode
4. ✅ Performance and reliability are maintained
5. ✅ No breaking changes exist

## Prerequisites

Before starting verification, ensure you have:

```bash
# 1. All services built
make build-all

# 2. Clean environment
pkill -f plantd || true

# 3. Current working directory is project root
cd /path/to/plantd
```

## 🚀 Quick Verification

For rapid verification, use the automated test script:

```bash
# Run comprehensive verification
./scripts/test-migration-verification.sh

# Quick health check only
./scripts/test-migration-verification.sh --quick

# Test specific functionality
./scripts/test-migration-verification.sh --auth-only
./scripts/test-migration-verification.sh --state-only
./scripts/test-migration-verification.sh --broker-only
```

## 📋 Manual Testing Procedures

### Step 1: Service Startup Testing

#### 1.1 Start Services with Dual Protocol Support

```bash
# Terminal 1: State Service (MDP + gRPC)
PLANTD_STATE_ENABLE_MDP=true PLANTD_STATE_ENABLE_GRPC=true PLANTD_STATE_GRPC_PORT=9080 ./build/plantd-state

# Terminal 2: Identity Service (MDP + gRPC)  
PLANTD_IDENTITY_ENABLE_MDP=true PLANTD_IDENTITY_ENABLE_GRPC=true PLANTD_IDENTITY_GRPC_PORT=8090 ./build/plantd-identity

# Terminal 3: Broker Service (MDP + gRPC)
PLANTD_BROKER_ENABLE_MDP=true PLANTD_BROKER_ENABLE_GRPC=true PLANTD_BROKER_GRPC_PORT=8082 ./build/plantd-broker
```

#### 1.2 Verify Service Health

```bash
# Check all health endpoints
curl -s http://localhost:9080/health | jq .    # State service
curl -s http://localhost:8090/health | jq .    # Identity service  
curl -s http://localhost:8082/health | jq .    # Broker service

# Check gRPC health endpoints
curl -s -X POST -H "Content-Type: application/json" -d '{}' \
  http://localhost:9080/plantd.state.v1.StateService/Health | jq .

curl -s -X POST -H "Content-Type: application/json" -d '{}' \
  http://localhost:8090/plantd.identity.v1.IdentityService/Health | jq .

curl -s -X POST -H "Content-Type: application/json" -d '{}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/Health | jq .
```

**Expected Results:**
- ✅ All services return HTTP 200 status
- ✅ Health responses show "healthy" or "SERVING" status
- ✅ Services show both MDP and gRPC transports active in logs

---

### Step 2: CLI Command Equivalence Testing

#### 2.1 Authentication Commands

Test that both protocols support identical authentication functionality:

```bash
# MDP Authentication
./build/plant auth status
./build/plant auth login --email="test@example.com" --password="password123"
./build/plant auth logout
./build/plant auth refresh

# gRPC Authentication (method 1: flag)
./build/plant auth status --use-grpc
./build/plant auth login --use-grpc --email="test@example.com" --password="password123"
./build/plant auth logout --use-grpc
./build/plant auth refresh --use-grpc

# gRPC Authentication (method 2: explicit commands)
./build/plant auth-grpc status --grpc-endpoint=http://localhost:8090
./build/plant auth-grpc login --grpc-endpoint=http://localhost:8090 --email="test@example.com" --password="password123"
./build/plant auth-grpc logout --grpc-endpoint=http://localhost:8090
./build/plant auth-grpc refresh --grpc-endpoint=http://localhost:8090
```

**Expected Results:**
- ✅ All command variants work identically
- ✅ Error messages are consistent between protocols
- ✅ Token management works with both protocols

#### 2.2 State Management Commands

Test all state operations with both protocols:

```bash
# Set test variables
TEST_SERVICE="org.plantd.TestService$(date +%s)"
TEST_KEY="test-key"
TEST_VALUE="test-value-$(date +%s)"

# MDP State Operations
./build/plant state list-scopes --service="$TEST_SERVICE"
./build/plant state set --service="$TEST_SERVICE" "$TEST_KEY" "$TEST_VALUE"
./build/plant state get --service="$TEST_SERVICE" "$TEST_KEY"
./build/plant state list --service="$TEST_SERVICE"
./build/plant state delete --service="$TEST_SERVICE" "$TEST_KEY"

# gRPC State Operations (method 1: flag)
./build/plant state list-scopes --use-grpc --service="$TEST_SERVICE"
./build/plant state set --use-grpc --service="$TEST_SERVICE" "$TEST_KEY" "$TEST_VALUE"
./build/plant state get --use-grpc --service="$TEST_SERVICE" "$TEST_KEY"
./build/plant state list --use-grpc --service="$TEST_SERVICE"
./build/plant state delete --use-grpc --service="$TEST_SERVICE" "$TEST_KEY"

# gRPC State Operations (method 2: explicit commands)
./build/plant state-grpc list-scopes --grpc-endpoint=http://localhost:9080 --service="$TEST_SERVICE"
./build/plant state-grpc set --grpc-endpoint=http://localhost:9080 --service="$TEST_SERVICE" "$TEST_KEY" "$TEST_VALUE"
./build/plant state-grpc get --grpc-endpoint=http://localhost:9080 --service="$TEST_SERVICE" "$TEST_KEY"
./build/plant state-grpc list --grpc-endpoint=http://localhost:9080 --service="$TEST_SERVICE"
./build/plant state-grpc delete --grpc-endpoint=http://localhost:9080 --service="$TEST_SERVICE" "$TEST_KEY"
```

**Expected Results:**
- ✅ All operations work identically across protocols
- ✅ Data consistency: values set via MDP are retrievable via gRPC and vice versa
- ✅ Error handling is consistent
- ✅ Performance is comparable

#### 2.3 Connectivity Testing

```bash
# Echo connectivity (if echo module available)
./build/plant echo "Hello PlantD MDP"
./build/plant echo "Hello PlantD gRPC" --use-grpc

# Configuration commands
./build/plant config show
./build/plant config validate

# Version and help commands
./build/plant version
./build/plant --help
./build/plant state --help
./build/plant auth --help
```

**Expected Results:**
- ✅ Echo commands work (if echo service available)
- ✅ Configuration commands show proper dual-protocol settings
- ✅ Help text shows all available commands and flags

---

### Step 3: Broker Service gRPC API Testing

#### 3.1 Service Registration and Discovery

```bash
# Register a test service
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"serviceId":"org.plantd.TestService","endpoint":"http://localhost:9999","capabilities":["test","demo"],"metadata":{"version":"1.0"}}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/RegisterService | jq .

# Get service information
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"serviceId":"org.plantd.TestService"}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/GetService | jq .

# List all services
curl -s -X POST -H "Content-Type: application/json" -d '{}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/ListServices | jq .
```

#### 3.2 Message Routing

```bash
# Route a message to registered service
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"targetService":"org.plantd.TestService","operation":"test","payload":"aGVsbG8gd29ybGQ=","headers":{"source":"test-client"}}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/RouteMessage | jq .

# Broadcast a message
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"servicePattern":"org.plantd.*","operation":"broadcast","payload":"YnJvYWRjYXN0IG1lc3NhZ2U=","headers":{"type":"broadcast"}}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/BroadcastMessage | jq .
```

#### 3.3 Bus Management

```bash
# Create a new message bus
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"name":"test-bus","type":"BUS_TYPE_CONTROL","config":{"description":"Test bus for verification"}}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/CreateBus | jq .

# List all buses
curl -s -X POST -H "Content-Type: application/json" -d '{}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/ListBuses | jq .

# Publish to bus
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"busName":"test-bus","topic":"test-topic","payload":"dGVzdCBtZXNzYWdl","headers":{"timestamp":"$(date -Iseconds)"}}' \
  http://localhost:8082/plantd.broker.v1.BrokerService/PublishToBus | jq .
```

**Expected Results:**
- ✅ Service registration returns valid registration ID
- ✅ Service discovery returns complete service information
- ✅ Message routing works correctly with proper headers
- ✅ Bus creation and management functions properly
- ✅ All responses have proper gRPC status codes

---

### Step 4: Protocol Consistency Verification

#### 4.1 Data Consistency Testing

Test that data operations are consistent across protocols:

```bash
# Test 1: Set via MDP, Get via gRPC
./build/plant state set --service="org.plantd.ConsistencyTest" "mdp-key" "mdp-value"
./build/plant state get --use-grpc --service="org.plantd.ConsistencyTest" "mdp-key"

# Test 2: Set via gRPC, Get via MDP  
./build/plant state set --use-grpc --service="org.plantd.ConsistencyTest" "grpc-key" "grpc-value"
./build/plant state get --service="org.plantd.ConsistencyTest" "grpc-key"

# Test 3: List operations should show data from both protocols
./build/plant state list --service="org.plantd.ConsistencyTest"
./build/plant state list --use-grpc --service="org.plantd.ConsistencyTest"
```

**Expected Results:**
- ✅ Data set via MDP is retrievable via gRPC
- ✅ Data set via gRPC is retrievable via MDP
- ✅ List operations show identical results regardless of protocol

#### 4.2 Error Handling Consistency

Test that error responses are consistent:

```bash
# Test authentication errors
./build/plant state get --service="org.plantd.Test" "nonexistent-key"
./build/plant state get --use-grpc --service="org.plantd.Test" "nonexistent-key"

# Test invalid operations
./build/plant state delete --service="org.plantd.Test" "nonexistent-key"
./build/plant state delete --use-grpc --service="org.plantd.Test" "nonexistent-key"
```

**Expected Results:**
- ✅ Error messages are consistent between protocols
- ✅ HTTP status codes and error types match
- ✅ Authentication failures behave identically

---

### Step 5: Performance and Load Testing

#### 5.1 Response Time Comparison

```bash
# Create test script for timing comparison
cat > /tmp/perf-test.sh << 'EOF'
#!/bin/bash
echo "=== MDP Performance ==="
time for i in {1..10}; do
  ./build/plant state set --service="org.plantd.PerfTest" "key$i" "value$i" > /dev/null
done

echo "=== gRPC Performance ==="
time for i in {1..10}; do  
  ./build/plant state set --use-grpc --service="org.plantd.PerfTest" "key$i" "value$i" > /dev/null
done
EOF

chmod +x /tmp/perf-test.sh
/tmp/perf-test.sh
```

#### 5.2 Concurrent Operations

```bash
# Test concurrent operations don't interfere
./build/plant state set --service="org.plantd.ConcurrentTest" "mdp-key" "mdp-value" &
./build/plant state set --use-grpc --service="org.plantd.ConcurrentTest" "grpc-key" "grpc-value" &
wait

# Verify both operations succeeded
./build/plant state get --service="org.plantd.ConcurrentTest" "mdp-key"
./build/plant state get --use-grpc --service="org.plantd.ConcurrentTest" "grpc-key"
```

**Expected Results:**
- ✅ gRPC performance is comparable to or better than MDP
- ✅ Concurrent operations work correctly
- ✅ No race conditions or data corruption

---

## 🏁 Migration Completion Criteria

The gRPC migration is considered **COMPLETE** and ready for MDP removal when:

### ✅ **Functional Completeness**
- [ ] All CLI commands work with both protocols
- [ ] Authentication flows are identical  
- [ ] State operations are fully equivalent
- [ ] Broker functionality is comprehensive
- [ ] Error handling is consistent

### ✅ **Data Integrity** 
- [ ] Cross-protocol data consistency verified
- [ ] No data loss during protocol switching
- [ ] Concurrent operations work correctly

### ✅ **Performance**
- [ ] gRPC response times ≤ MDP response times
- [ ] Memory usage is acceptable
- [ ] CPU usage is comparable or better

### ✅ **Reliability**
- [ ] Services start reliably with dual protocols
- [ ] Health checks work for all services
- [ ] Error recovery functions properly

### ✅ **Documentation**
- [ ] Migration testing is documented
- [ ] Performance benchmarks are recorded
- [ ] Known issues are documented

## 🎯 Decision Matrix

| Criteria | Status | Notes |
|----------|--------|-------|
| **CLI Functionality** | ⚪ Testing | All commands must work identically |
| **Service Health** | ⚪ Testing | All services must start and respond |  
| **Data Consistency** | ⚪ Testing | Cross-protocol data integrity required |
| **Performance** | ⚪ Testing | gRPC must meet or exceed MDP performance |
| **Error Handling** | ⚪ Testing | Consistent error responses required |
| **Documentation** | ⚪ Testing | Complete testing records required |

**Legend:** ✅ Complete | ⚪ Testing | ❌ Failed

## 🚨 Troubleshooting

### Common Issues

1. **Service Won't Start**
   ```bash
   # Check port conflicts
   lsof -i :9080 -i :8090 -i :8082
   
   # Check environment variables
   env | grep PLANTD
   ```

2. **gRPC Connection Refused**
   ```bash
   # Verify service is listening on correct port
   curl -v http://localhost:9080/health
   
   # Check firewall/network settings
   telnet localhost 9080
   ```

3. **Authentication Failures**
   ```bash
   # Clear token cache
   rm -rf ~/.config/plantd/auth/
   
   # Check identity service logs
   # Review authentication configuration
   ```

4. **Data Inconsistency**
   ```bash
   # Check service logs for errors
   # Verify both protocols connect to same backend
   # Test with fresh service instances
   ```

### Test Result Analysis

If tests fail:

1. **Review Error Logs**: Check `/tmp/plantd-migration-test-errors.log`
2. **Check Service Logs**: Review output from running services
3. **Verify Configuration**: Ensure environment variables are correct
4. **Test Incrementally**: Use specific test flags (`--auth-only`, etc.)
5. **Compare Protocols**: Run identical commands with both protocols

## 📊 Final Migration Report

After completing all tests, generate a migration report:

```bash
# Run comprehensive verification
./scripts/test-migration-verification.sh > migration-report.txt 2>&1

# Review results
cat migration-report.txt
less /tmp/plantd-migration-test-results.log
less /tmp/plantd-migration-test-errors.log
```

**If all tests pass**: ✅ **Ready to remove MDP dependencies**
**If tests fail**: ❌ **Address issues before removing MDP**

---

## 🎉 Next Steps

Once verification is complete and all tests pass:

1. **Document Results**: Save test reports and performance benchmarks
2. **Plan MDP Removal**: Create removal strategy for legacy code  
3. **Update Configuration**: Switch default protocol to gRPC
4. **Remove Dependencies**: Eliminate MDP/ZeroMQ dependencies
5. **Final Testing**: Verify system works with gRPC-only mode

**The comprehensive verification ensures zero-risk migration from MDP to gRPC!** 🚀 
