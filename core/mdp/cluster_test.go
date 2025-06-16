package mdp

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrokerNode(t *testing.T) {
	node := &BrokerNode{
		ID:           "broker-001",
		Endpoint:     "tcp://localhost:9797",
		LastSeen:     time.Now(),
		Status:       "active",
		Load:         5,
		Services:     []string{"echo", "calculator"},
		FailureCount: 0,
	}

	assert.Equal(t, "broker-001", node.ID)
	assert.Equal(t, "tcp://localhost:9797", node.Endpoint)
	assert.Equal(t, "active", node.Status)
	assert.Equal(t, 5, node.Load)
	assert.Equal(t, []string{"echo", "calculator"}, node.Services)
	assert.Equal(t, 0, node.FailureCount)
}

func TestClusterManager(t *testing.T) {
	config := ClusterConfig{
		LocalID:           "test-broker-001",
		LocalEndpoint:     "tcp://localhost:9797",
		DiscoveryEndpoint: "tcp://localhost:8888",
		HeartbeatInterval: 1 * time.Second,
		FailureThreshold:  3,
	}

	cm, err := NewClusterManager(config)
	require.NoError(t, err)
	defer cm.Stop()

	t.Run("InitialState", func(t *testing.T) {
		assert.Equal(t, config.LocalID, cm.localNode.ID)
		assert.Equal(t, config.LocalEndpoint, cm.localNode.Endpoint)
		assert.Equal(t, "active", cm.localNode.Status)
		assert.Equal(t, 0, cm.localNode.Load)
		assert.Empty(t, cm.localNode.Services)

		// Local node should be in the cluster
		nodes := cm.GetNodes()
		assert.Len(t, nodes, 1)
		assert.Contains(t, nodes, config.LocalID)
	})

	t.Run("StartAndStop", func(t *testing.T) {
		err := cm.Start()
		assert.NoError(t, err)

		// Give some time for goroutines to start
		time.Sleep(100 * time.Millisecond)

		err = cm.Stop()
		assert.NoError(t, err)
	})

	t.Run("UpdateLocalLoad", func(t *testing.T) {
		services := []string{"echo", "calculator", "state"}
		cm.UpdateLocalLoad(10, services)

		assert.Equal(t, 10, cm.localNode.Load)
		assert.Equal(t, services, cm.localNode.Services)
	})

	t.Run("AddNode", func(t *testing.T) {
		newNode := &BrokerNode{
			ID:       "broker-002",
			Endpoint: "tcp://localhost:9798",
			Load:     3,
			Services: []string{"echo"},
		}

		cm.AddNode(newNode)

		nodes := cm.GetNodes()
		assert.Len(t, nodes, 2)
		assert.Contains(t, nodes, "broker-002")
		assert.Equal(t, "active", nodes["broker-002"].Status)
	})

	t.Run("GetActiveNodes", func(t *testing.T) {
		// Add another node
		activeNode := &BrokerNode{
			ID:       "broker-003",
			Endpoint: "tcp://localhost:9799",
			Status:   "active",
			Load:     2,
		}
		cm.AddNode(activeNode)

		// Add a failed node
		failedNode := &BrokerNode{
			ID:       "broker-004",
			Endpoint: "tcp://localhost:9800",
			Status:   "failed",
			Load:     0,
		}
		cm.AddNode(failedNode)

		activeNodes := cm.GetActiveNodes()
		assert.Len(t, activeNodes, 3) // local + broker-002 + broker-003 (all active)

		var activeIDs []string
		for _, node := range activeNodes {
			activeIDs = append(activeIDs, node.ID)
		}
		assert.Contains(t, activeIDs, "test-broker-001")
		assert.Contains(t, activeIDs, "broker-002")
		assert.Contains(t, activeIDs, "broker-003")
		assert.NotContains(t, activeIDs, "broker-004")
	})

	t.Run("GetBestBroker", func(t *testing.T) {
		// Clear nodes and add test nodes with different loads
		cm.Stop()
		cm, _ = NewClusterManager(config)

		cm.AddNode(&BrokerNode{
			ID:       "high-load",
			Endpoint: "tcp://localhost:9801",
			Status:   "active",
			Load:     20,
		})

		cm.AddNode(&BrokerNode{
			ID:       "low-load",
			Endpoint: "tcp://localhost:9802",
			Status:   "active",
			Load:     2,
		})

		cm.AddNode(&BrokerNode{
			ID:       "medium-load",
			Endpoint: "tcp://localhost:9803",
			Status:   "active",
			Load:     10,
		})

		bestBroker := cm.GetBestBroker()
		assert.NotNil(t, bestBroker)
		assert.Equal(t, "test-broker-001", bestBroker.ID) // Local node has load 0
	})

	t.Run("GetBrokerForService", func(t *testing.T) {
		// Clear and setup test scenario
		cm.Stop()
		cm, _ = NewClusterManager(config)

		// Node with calculator service
		cm.AddNode(&BrokerNode{
			ID:       "calc-node",
			Endpoint: "tcp://localhost:9804",
			Status:   "active",
			Load:     5,
			Services: []string{"calculator", "math"},
		})

		// Node with echo service
		cm.AddNode(&BrokerNode{
			ID:       "echo-node",
			Endpoint: "tcp://localhost:9805",
			Status:   "active",
			Load:     3,
			Services: []string{"echo", "ping"},
		})

		// Test service-specific routing
		calcBroker := cm.GetBrokerForService("calculator")
		assert.NotNil(t, calcBroker)
		assert.Equal(t, "calc-node", calcBroker.ID)

		echoBroker := cm.GetBrokerForService("echo")
		assert.NotNil(t, echoBroker)
		assert.Equal(t, "echo-node", echoBroker.ID)

		// Test unknown service - should return best available
		unknownBroker := cm.GetBrokerForService("unknown")
		assert.NotNil(t, unknownBroker)
		assert.Equal(t, "test-broker-001", unknownBroker.ID) // Local node has lowest load (0)
	})

	t.Run("RemoveNode", func(t *testing.T) {
		initialCount := len(cm.GetNodes())

		cm.RemoveNode("echo-node")

		nodes := cm.GetNodes()
		assert.Len(t, nodes, initialCount-1)
		assert.NotContains(t, nodes, "echo-node")
	})

	t.Run("GetClusterStats", func(t *testing.T) {
		stats := cm.GetClusterStats()

		assert.Contains(t, stats, "total_nodes")
		assert.Contains(t, stats, "status_breakdown")
		assert.Contains(t, stats, "total_load")
		assert.Contains(t, stats, "service_distribution")
		assert.Contains(t, stats, "local_node_id")

		assert.Equal(t, config.LocalID, stats["local_node_id"])
		assert.IsType(t, 0, stats["total_nodes"])
		assert.IsType(t, make(map[string]int), stats["status_breakdown"])
	})

	t.Run("NodeCallbacks", func(t *testing.T) {
		callbackExecuted := false

		cm.OnNodeUpdate(func(node *BrokerNode) {
			callbackExecuted = true
		})

		// Add a node to trigger callback
		testNode := &BrokerNode{
			ID:       "callback-test",
			Endpoint: "tcp://localhost:9806",
			Status:   "active",
			Load:     1,
		}
		cm.AddNode(testNode)

		// Callbacks are called asynchronously, so wait a bit
		time.Sleep(10 * time.Millisecond)

		// Note: Callbacks are only triggered in processHeartbeat, not AddNode
		// This test structure shows how callbacks would work
		assert.False(t, callbackExecuted) // AddNode doesn't trigger callbacks
	})
}

func TestClusterFailureDetection(t *testing.T) {
	config := ClusterConfig{
		LocalID:           "test-broker",
		LocalEndpoint:     "tcp://localhost:9797",
		DiscoveryEndpoint: "tcp://localhost:8888",
		HeartbeatInterval: 100 * time.Millisecond,
		FailureThreshold:  3,
	}

	cm, err := NewClusterManager(config)
	require.NoError(t, err)
	defer cm.Stop()

	// Add a node with old timestamp to simulate failure
	oldNode := &BrokerNode{
		ID:       "old-node",
		Endpoint: "tcp://localhost:9801",
		Status:   "active",
		Load:     5,
		LastSeen: time.Now().Add(-2 * time.Minute), // 2 minutes ago
	}
	cm.AddNode(oldNode)

	// Manually trigger health check
	cm.checkNodeHealth()

	nodes := cm.GetNodes()
	assert.Equal(t, "failed", nodes["old-node"].Status)
	assert.Equal(t, 1, nodes["old-node"].FailureCount)
}

func TestLoadBalancer(t *testing.T) {
	config := ClusterConfig{
		LocalID:           "lb-test-broker",
		LocalEndpoint:     "tcp://localhost:9797",
		DiscoveryEndpoint: "tcp://localhost:8888",
		HeartbeatInterval: 1 * time.Second,
		FailureThreshold:  3,
	}

	cm, err := NewClusterManager(config)
	require.NoError(t, err)
	defer cm.Stop()

	// Setup test nodes
	cm.AddNode(&BrokerNode{
		ID:       "service-a",
		Endpoint: "tcp://localhost:9801",
		Status:   "active",
		Load:     10,
		Services: []string{"service-a", "common"},
	})

	cm.AddNode(&BrokerNode{
		ID:       "service-b",
		Endpoint: "tcp://localhost:9802",
		Status:   "active",
		Load:     5,
		Services: []string{"service-b", "common"},
	})

	cm.AddNode(&BrokerNode{
		ID:       "low-load",
		Endpoint: "tcp://localhost:9803",
		Status:   "active",
		Load:     1,
		Services: []string{"other"},
	})

	t.Run("LeastLoadStrategy", func(t *testing.T) {
		lb := NewLoadBalancer(cm, LeastLoad)

		broker := lb.SelectBroker("any-service")
		assert.NotNil(t, broker)
		assert.Equal(t, "lb-test-broker", broker.ID) // Local node has load 0
	})

	t.Run("ServiceAwareStrategy", func(t *testing.T) {
		lb := NewLoadBalancer(cm, ServiceAware)

		// Test service-specific selection
		brokerA := lb.SelectBroker("service-a")
		assert.NotNil(t, brokerA)
		assert.Equal(t, "service-a", brokerA.ID)

		// Test common service selection - should pick lower load
		commonBroker := lb.SelectBroker("common")
		assert.NotNil(t, commonBroker)
		assert.Equal(t, "service-b", commonBroker.ID) // Lower load (5 vs 10)

		// Test unknown service - should fallback to least load
		unknownBroker := lb.SelectBroker("unknown")
		assert.NotNil(t, unknownBroker)
		assert.Equal(t, "lb-test-broker", unknownBroker.ID) // Local node has load 0
	})

	t.Run("GetLoadDistribution", func(t *testing.T) {
		lb := NewLoadBalancer(cm, LeastLoad)

		distribution := lb.GetLoadDistribution()
		assert.Contains(t, distribution, "lb-test-broker")
		assert.Contains(t, distribution, "service-a")
		assert.Contains(t, distribution, "service-b")
		assert.Contains(t, distribution, "low-load")

		assert.Equal(t, 0, distribution["lb-test-broker"])
		assert.Equal(t, 10, distribution["service-a"])
		assert.Equal(t, 5, distribution["service-b"])
		assert.Equal(t, 1, distribution["low-load"])
	})
}

func TestClusterIntegration(t *testing.T) {
	// Test integration between clustering and load balancing

	config1 := ClusterConfig{
		LocalID:           "cluster-broker-1",
		LocalEndpoint:     "tcp://localhost:9797",
		DiscoveryEndpoint: "tcp://localhost:8881",
		HeartbeatInterval: 500 * time.Millisecond,
		FailureThreshold:  3,
	}

	config2 := ClusterConfig{
		LocalID:           "cluster-broker-2",
		LocalEndpoint:     "tcp://localhost:9798",
		DiscoveryEndpoint: "tcp://localhost:8882",
		HeartbeatInterval: 500 * time.Millisecond,
		FailureThreshold:  3,
	}

	cm1, err := NewClusterManager(config1)
	require.NoError(t, err)
	defer cm1.Stop()

	cm2, err := NewClusterManager(config2)
	require.NoError(t, err)
	defer cm2.Stop()

	// Simulate cross-cluster awareness (in real implementation, this would be automatic)
	cm1.AddNode(&BrokerNode{
		ID:       config2.LocalID,
		Endpoint: config2.LocalEndpoint,
		Status:   "active",
		Load:     5,
		Services: []string{"remote-service"},
	})

	cm2.AddNode(&BrokerNode{
		ID:       config1.LocalID,
		Endpoint: config1.LocalEndpoint,
		Status:   "active",
		Load:     3,
		Services: []string{"local-service"},
	})

	// Update loads
	cm1.UpdateLocalLoad(2, []string{"local-service", "shared"})
	cm2.UpdateLocalLoad(5, []string{"remote-service", "shared"})

	// Test load balancing across cluster
	lb1 := NewLoadBalancer(cm1, ServiceAware)

	// From CM1's perspective, selecting broker for remote-service
	remoteBroker := lb1.SelectBroker("remote-service")
	assert.NotNil(t, remoteBroker)
	assert.Equal(t, "cluster-broker-2", remoteBroker.ID)

	// From CM1's perspective, selecting broker for local-service
	localBroker := lb1.SelectBroker("local-service")
	assert.NotNil(t, localBroker)
	assert.Equal(t, "cluster-broker-1", localBroker.ID)

	// Test cluster stats
	stats1 := cm1.GetClusterStats()
	assert.Equal(t, 2, stats1["total_nodes"])
	assert.Equal(t, "cluster-broker-1", stats1["local_node_id"])

	distribution1 := lb1.GetLoadDistribution()
	assert.Equal(t, 2, distribution1["cluster-broker-1"])
	assert.Equal(t, 5, distribution1["cluster-broker-2"])
}

func BenchmarkClusterOperations(b *testing.B) {
	config := ClusterConfig{
		LocalID:           "bench-broker",
		LocalEndpoint:     "tcp://localhost:9797",
		DiscoveryEndpoint: "tcp://localhost:8888",
		HeartbeatInterval: 1 * time.Second,
		FailureThreshold:  3,
	}

	cm, _ := NewClusterManager(config)
	defer cm.Stop()

	// Add some nodes for realistic benchmarking
	for i := 0; i < 10; i++ {
		cm.AddNode(&BrokerNode{
			ID:       fmt.Sprintf("bench-node-%d", i),
			Endpoint: fmt.Sprintf("tcp://localhost:%d", 9800+i),
			Status:   "active",
			Load:     i * 2,
			Services: []string{fmt.Sprintf("service-%d", i%3)},
		})
	}

	lb := NewLoadBalancer(cm, ServiceAware)

	b.ResetTimer()

	b.Run("GetBestBroker", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cm.GetBestBroker()
		}
	})

	b.Run("GetBrokerForService", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cm.GetBrokerForService("service-1")
		}
	})

	b.Run("LoadBalancerSelect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = lb.SelectBroker("service-2")
		}
	})

	b.Run("UpdateLocalLoad", func(b *testing.B) {
		services := []string{"service-1", "service-2"}
		for i := 0; i < b.N; i++ {
			cm.UpdateLocalLoad(i%20, services)
		}
	})

	b.Run("GetClusterStats", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cm.GetClusterStats()
		}
	})
}
