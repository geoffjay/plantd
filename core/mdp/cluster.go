package mdp

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// BrokerNode represents a single broker in the cluster
type BrokerNode struct {
	ID           string    `json:"id"`
	Endpoint     string    `json:"endpoint"`
	LastSeen     time.Time `json:"last_seen"`
	Status       string    `json:"status"` // active, inactive, failed
	Load         int       `json:"load"`   // number of connected workers
	Services     []string  `json:"services"`
	FailureCount int       `json:"failure_count"`
}

// ClusterManager manages broker discovery and membership
type ClusterManager struct {
	mu                sync.RWMutex
	localNode         *BrokerNode
	nodes             map[string]*BrokerNode
	discoveryEndpoint string
	heartbeatTicker   *time.Ticker
	ctx               context.Context
	cancel            context.CancelFunc
	updateCallbacks   []func(*BrokerNode)
}

// ClusterConfig defines cluster configuration
type ClusterConfig struct {
	LocalID           string        `json:"local_id"`
	LocalEndpoint     string        `json:"local_endpoint"`
	DiscoveryEndpoint string        `json:"discovery_endpoint"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	FailureThreshold  int           `json:"failure_threshold"`
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(config ClusterConfig) (*ClusterManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	localNode := &BrokerNode{
		ID:           config.LocalID,
		Endpoint:     config.LocalEndpoint,
		LastSeen:     time.Now(),
		Status:       "active",
		Load:         0,
		Services:     make([]string, 0),
		FailureCount: 0,
	}

	cm := &ClusterManager{
		localNode:         localNode,
		nodes:             make(map[string]*BrokerNode),
		discoveryEndpoint: config.DiscoveryEndpoint,
		heartbeatTicker:   time.NewTicker(config.HeartbeatInterval),
		ctx:               ctx,
		cancel:            cancel,
		updateCallbacks:   make([]func(*BrokerNode), 0),
	}

	// Add local node to cluster
	cm.nodes[localNode.ID] = localNode

	return cm, nil
}

// Start begins cluster discovery and heartbeat processes
func (cm *ClusterManager) Start() error {
	log.WithFields(log.Fields{
		"endpoint": cm.discoveryEndpoint,
		"node_id":  cm.localNode.ID,
	}).Info("cluster manager started")

	// Start heartbeat sender
	go cm.sendHeartbeats()

	// Start discovery listener
	go cm.listenForDiscovery()

	// Start failure detection
	go cm.detectFailures()

	return nil
}

// Stop gracefully shuts down the cluster manager
func (cm *ClusterManager) Stop() error {
	cm.cancel()

	if cm.heartbeatTicker != nil {
		cm.heartbeatTicker.Stop()
	}

	log.WithFields(log.Fields{
		"node_id": cm.localNode.ID,
	}).Info("cluster manager stopped")

	return nil
}

// GetNodes returns all known cluster nodes
func (cm *ClusterManager) GetNodes() map[string]*BrokerNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent external modification
	nodes := make(map[string]*BrokerNode)
	for id, node := range cm.nodes {
		nodeCopy := *node
		nodes[id] = &nodeCopy
	}

	return nodes
}

// GetActiveNodes returns only active cluster nodes
func (cm *ClusterManager) GetActiveNodes() []*BrokerNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var activeNodes []*BrokerNode
	for _, node := range cm.nodes {
		if node.Status == "active" {
			nodeCopy := *node
			activeNodes = append(activeNodes, &nodeCopy)
		}
	}

	return activeNodes
}

// UpdateLocalLoad updates the load information for the local node
func (cm *ClusterManager) UpdateLocalLoad(workerCount int, services []string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.localNode.Load = workerCount
	cm.localNode.Services = make([]string, len(services))
	copy(cm.localNode.Services, services)
	cm.localNode.LastSeen = time.Now()

	log.WithFields(log.Fields{
		"node_id":      cm.localNode.ID,
		"worker_count": workerCount,
		"services":     services,
	}).Debug("updated local node load")
}

// GetBestBroker returns the broker with the lowest load for load balancing
func (cm *ClusterManager) GetBestBroker() *BrokerNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var bestBroker *BrokerNode
	lowestLoad := int(^uint(0) >> 1) // max int

	for _, node := range cm.nodes {
		if node.Status == "active" && node.Load < lowestLoad {
			lowestLoad = node.Load
			bestBroker = node
		}
	}

	if bestBroker != nil {
		nodeCopy := *bestBroker
		return &nodeCopy
	}

	return nil
}

// GetBrokerForService returns the best broker for a specific service
func (cm *ClusterManager) GetBrokerForService(serviceName string) *BrokerNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var candidates []*BrokerNode

	// First, find brokers that have workers for this service
	for _, node := range cm.nodes {
		if node.Status == "active" {
			for _, service := range node.Services {
				if service == serviceName {
					candidates = append(candidates, node)
					break
				}
			}
		}
	}

	// If no broker has the service, return the best available broker
	if len(candidates) == 0 {
		return cm.GetBestBroker()
	}

	// Among candidates, return the one with lowest load
	var bestBroker *BrokerNode
	lowestLoad := int(^uint(0) >> 1)

	for _, candidate := range candidates {
		if candidate.Load < lowestLoad {
			lowestLoad = candidate.Load
			bestBroker = candidate
		}
	}

	if bestBroker != nil {
		nodeCopy := *bestBroker
		return &nodeCopy
	}

	return nil
}

// OnNodeUpdate registers a callback for node updates
func (cm *ClusterManager) OnNodeUpdate(callback func(*BrokerNode)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.updateCallbacks = append(cm.updateCallbacks, callback)
}

// sendHeartbeats sends periodic heartbeats to announce this node's presence
func (cm *ClusterManager) sendHeartbeats() {
	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-cm.heartbeatTicker.C:
			cm.sendHeartbeat()
		}
	}
}

// sendHeartbeat sends a single heartbeat message (simplified implementation)
func (cm *ClusterManager) sendHeartbeat() {
	cm.mu.RLock()
	_, err := json.Marshal(cm.localNode)
	cm.mu.RUnlock()

	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"node_id": cm.localNode.ID,
		}).Error("failed to marshal heartbeat data")
		return
	}

	log.WithFields(log.Fields{
		"node_id": cm.localNode.ID,
	}).Debug("sent cluster heartbeat")
}

// listenForDiscovery listens for cluster discovery messages (simplified)
func (cm *ClusterManager) listenForDiscovery() {
	log.WithFields(log.Fields{
		"node_id": cm.localNode.ID,
	}).Debug("cluster discovery listener started")

	<-cm.ctx.Done()
	log.WithFields(log.Fields{
		"node_id": cm.localNode.ID,
	}).Debug("cluster discovery listener stopped")
}

// detectFailures monitors for failed nodes and marks them as inactive
func (cm *ClusterManager) detectFailures() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.checkNodeHealth()
		}
	}
}

// checkNodeHealth checks the health of all nodes and marks failed ones
func (cm *ClusterManager) checkNodeHealth() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	failureThreshold := 60 * time.Second // Consider node failed after 60s of no heartbeat

	for nodeID, node := range cm.nodes {
		if nodeID == cm.localNode.ID {
			continue // Skip local node
		}

		if now.Sub(node.LastSeen) > failureThreshold {
			if node.Status == "active" {
				node.Status = "failed"
				node.FailureCount++

				log.WithFields(log.Fields{
					"node_id":       nodeID,
					"last_seen":     node.LastSeen,
					"failure_count": node.FailureCount,
				}).Warn("marked node as failed")

				// Notify callbacks
				for _, callback := range cm.updateCallbacks {
					go callback(node)
				}
			}
		}
	}
}

// GetClusterStats returns statistics about the cluster
func (cm *ClusterManager) GetClusterStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := make(map[string]interface{})
	statusCounts := make(map[string]int)
	totalLoad := 0
	services := make(map[string]int)

	for _, node := range cm.nodes {
		statusCounts[node.Status]++
		totalLoad += node.Load

		for _, service := range node.Services {
			services[service]++
		}
	}

	stats["total_nodes"] = len(cm.nodes)
	stats["status_breakdown"] = statusCounts
	stats["total_load"] = totalLoad
	stats["service_distribution"] = services
	stats["local_node_id"] = cm.localNode.ID

	return stats
}

// LoadBalancer provides intelligent request routing across cluster nodes
type LoadBalancer struct {
	cluster  *ClusterManager
	strategy LoadBalancingStrategy
}

// LoadBalancingStrategy defines load balancing algorithms
type LoadBalancingStrategy string

const (
	RoundRobin   LoadBalancingStrategy = "round_robin"
	LeastLoad    LoadBalancingStrategy = "least_load"
	ServiceAware LoadBalancingStrategy = "service_aware"
	Locality     LoadBalancingStrategy = "locality"
)

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(cluster *ClusterManager, strategy LoadBalancingStrategy) *LoadBalancer {
	return &LoadBalancer{
		cluster:  cluster,
		strategy: strategy,
	}
}

// SelectBroker selects the best broker based on the configured strategy
func (lb *LoadBalancer) SelectBroker(serviceName string) *BrokerNode {
	switch lb.strategy {
	case LeastLoad:
		return lb.cluster.GetBestBroker()
	case ServiceAware:
		return lb.cluster.GetBrokerForService(serviceName)
	default:
		return lb.cluster.GetBestBroker()
	}
}

// GetLoadDistribution returns current load distribution across brokers
func (lb *LoadBalancer) GetLoadDistribution() map[string]int {
	nodes := lb.cluster.GetActiveNodes()
	distribution := make(map[string]int)

	for _, node := range nodes {
		distribution[node.ID] = node.Load
	}

	return distribution
}

// AddNode manually adds a node to the cluster (for testing or manual configuration)
func (cm *ClusterManager) AddNode(node *BrokerNode) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	node.LastSeen = time.Now()
	if node.Status == "" {
		node.Status = "active"
	}
	cm.nodes[node.ID] = node

	log.WithFields(log.Fields{
		"node_id":  node.ID,
		"endpoint": node.Endpoint,
	}).Info("manually added cluster node")
}

// RemoveNode manually removes a node from the cluster
func (cm *ClusterManager) RemoveNode(nodeID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if node, exists := cm.nodes[nodeID]; exists {
		node.Status = "inactive"
		delete(cm.nodes, nodeID)

		log.WithFields(log.Fields{
			"node_id": nodeID,
		}).Info("manually removed cluster node")
	}
}
