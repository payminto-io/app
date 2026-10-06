package blockchain

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// RPCPool manages a set of RPC endpoints for one blockchain. It picks a
// healthy node round-robin, tracks fail counts, and temporarily evicts
// unhealthy nodes until a cooldown expires.
//
// Usage:
//
//	pool := NewRPCPool(blockchainID, rpcNodeRepo)
//	if err := pool.Refresh(); err != nil { ... }
//	node, err := pool.Pick()
//	// ... call RPC via node.URL ...
//	if err != nil { pool.MarkFailure(node.ID, err) }
//	else { pool.MarkSuccess(node.ID) }
type RPCPool struct {
	blockchainID uint
	repo         repository.RPCNodeRepository

	mu            sync.RWMutex
	nodes         []models.RPCNode
	roundRobinIdx atomic.Uint64
	cooldown      time.Duration
	failThreshold int
}

// NewRPCPool creates an unpopulated pool for the given blockchainID backed by
// the provided RPCNodeRepository. Call Refresh() to load nodes from the database.
func NewRPCPool(blockchainID uint, repo repository.RPCNodeRepository) *RPCPool {
	return &RPCPool{
		blockchainID:  blockchainID,
		repo:          repo,
		cooldown:      5 * time.Minute,
		failThreshold: 3,
	}
}

// Len returns the number of nodes in the pool.
func (p *RPCPool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.nodes)
}

// WithCooldown sets how long an unhealthy node stays out of rotation before
// HealthCheck will attempt to restore it. Returns the pool for chaining.
func (p *RPCPool) WithCooldown(d time.Duration) *RPCPool {
	p.cooldown = d
	return p
}

// WithFailThreshold sets how many consecutive failures are required before a
// node is marked unhealthy and evicted from the pool. Returns the pool for
// chaining.
func (p *RPCPool) WithFailThreshold(n int) *RPCPool {
	p.failThreshold = n
	return p
}

// Refresh reloads the pool's node list from the database, replacing the
// current set of healthy nodes and resetting the round-robin counter to zero.
func (p *RPCPool) Refresh() error {
	nodes, err := p.repo.ListHealthyByBlockchain(p.blockchainID)
	if err != nil {
		return fmt.Errorf("list rpc nodes: %w", err)
	}
	p.mu.Lock()
	p.nodes = nodes
	p.roundRobinIdx.Store(0)
	p.mu.Unlock()
	return nil
}

// Size returns how many nodes are currently in the pool.
func (p *RPCPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.nodes)
}

// Pick returns the next healthy node using round-robin selection. It is safe
// to call concurrently from multiple goroutines: the node slice is read under
// a shared (RLock) lock while the round-robin counter advances atomically,
// eliminating contention between concurrent callers. Returns an error if the
// pool contains no nodes.
func (p *RPCPool) Pick() (*models.RPCNode, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.nodes) == 0 {
		return nil, errors.New("rpc pool empty")
	}
	idx := p.roundRobinIdx.Add(1) - 1
	n := p.nodes[idx%uint64(len(p.nodes))]
	return &n, nil
}

// MarkSuccess resets the fail count for nodeID to zero in the database and
// updates its last_health_check timestamp.
func (p *RPCPool) MarkSuccess(nodeID uint) {
	_ = p.repo.ResetFailCount(nodeID)
	_ = p.repo.TouchLastHealthCheck(nodeID)
}

// MarkFailure increments the consecutive failure count for nodeID. If the
// count reaches the configured fail threshold the node is marked unhealthy in
// the database and evicted from the in-memory pool so no further picks are
// routed to it until HealthCheck restores it.
func (p *RPCPool) MarkFailure(nodeID uint, cause error) {
	_ = p.repo.IncrementFailCount(nodeID)

	node, err := p.repo.GetByID(nodeID)
	if err != nil {
		return
	}
	if node.FailCount >= p.failThreshold {
		reason := "unknown"
		if cause != nil {
			reason = cause.Error()
		}
		_ = p.repo.MarkUnhealthy(nodeID, reason)
		p.evictFromMemory(nodeID)
	}
}

// HealthCheck inspects all nodes for the pool's blockchain. Any node that is
// currently unhealthy and whose last error occurred longer ago than the
// configured cooldown is promoted back to healthy in the database. Refresh is
// then called so the restored nodes re-enter the in-memory rotation.
func (p *RPCPool) HealthCheck() error {
	allNodes, err := p.repo.ListByBlockchain(p.blockchainID)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, n := range allNodes {
		if n.Status == models.RPCNodeStatusUnhealthy && n.LastErrorAt != nil {
			if now.Sub(*n.LastErrorAt) >= p.cooldown {
				_ = p.repo.MarkHealthy(n.ID)
			}
		}
	}
	// Reload the pool to pick up newly healthy nodes.
	return p.Refresh()
}

func (p *RPCPool) evictFromMemory(nodeID uint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nodes = slices.DeleteFunc(p.nodes, func(n models.RPCNode) bool {
		return n.ID == nodeID
	})
}
