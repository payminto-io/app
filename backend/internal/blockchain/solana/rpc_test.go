package solana

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func rpcServer(t *testing.T, handler func(method string, n int) (int, any)) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		n := int(atomic.AddInt32(&calls, 1))
		status, result := handler(req.Method, n)
		w.WriteHeader(status)
		if status == http.StatusOK {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func testPool(t *testing.T, urls ...string) (*blockchain.RPCPool, repository.RPCNodeRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.RPCNode{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRPCNodeRepository(db)
	for i, u := range urls {
		if err := repo.Create(&models.RPCNode{BlockchainID: 1, Name: "n", URL: u, Priority: i, Status: models.RPCNodeStatusHealthy}); err != nil {
			t.Fatal(err)
		}
	}
	pool := blockchain.NewRPCPool(1, repo).WithFailThreshold(1)
	if err := pool.Refresh(); err != nil {
		t.Fatal(err)
	}
	return pool, repo
}

// L6/I5: a 429 must back off and try another node, never mark the node unhealthy.
func TestPoolCaller_RateLimitedNodeIsNotMarkedUnhealthy(t *testing.T) {
	limited, limitedCalls := rpcServer(t, func(string, int) (int, any) { return http.StatusTooManyRequests, nil })
	healthy, healthyCalls := rpcServer(t, func(string, int) (int, any) { return http.StatusOK, 42 })
	pool, repo := testPool(t, limited.URL, healthy.URL)
	caller := NewPoolCaller(pool)
	caller.backoffBase = time.Millisecond
	var slot uint64
	for i := 0; i < 4; i++ {
		if err := caller.Call(context.Background(), "getSlot", nil, &slot); err != nil || slot != 42 {
			t.Fatalf("call %d: %v slot=%d", i, err, slot)
		}
	}
	if pool.Len() != 2 {
		t.Fatalf("pool evicted a node: %d left", pool.Len())
	}
	nodes, _ := repo.ListByBlockchain(1)
	for _, n := range nodes {
		if n.Status != models.RPCNodeStatusHealthy {
			t.Fatalf("node %s marked %s after 429", n.URL, n.Status)
		}
	}
	if atomic.LoadInt32(healthyCalls) < 4 || atomic.LoadInt32(limitedCalls) < 1 {
		t.Fatalf("calls: limited=%d healthy=%d", *limitedCalls, *healthyCalls)
	}
}

func TestPoolCaller_PerNodeRateLimitSpacesCalls(t *testing.T) {
	srv, calls := rpcServer(t, func(string, int) (int, any) { return http.StatusOK, 1 })
	pool, _ := testPool(t, srv.URL)
	caller := NewPoolCaller(pool).WithRateLimit(50)
	start := time.Now()
	for i := 0; i < 6; i++ {
		var out uint64
		if err := caller.Call(context.Background(), "getSlot", nil, &out); err != nil {
			t.Fatal(err)
		}
	}
	if atomic.LoadInt32(calls) != 6 {
		t.Fatalf("calls = %d", *calls)
	}
	// 50 per second with a burst of one: six calls take at least ~100ms.
	if time.Since(start) < 90*time.Millisecond {
		t.Fatalf("calls were not rate limited: %s", time.Since(start))
	}
}

// C1: a transaction unknown to one node is fetched from another before giving up.
func TestClient_GetTransactionFromNodesTriesOtherNodes(t *testing.T) {
	lagging, _ := rpcServer(t, func(string, int) (int, any) { return http.StatusOK, nil })
	current, _ := rpcServer(t, func(string, int) (int, any) {
		return http.StatusOK, map[string]any{"slot": 7, "transaction": map[string]any{"signatures": []string{"sig"}}, "meta": map[string]any{"err": nil, "fee": 5000}}
	})
	pool, _ := testPool(t, lagging.URL, current.URL)
	c := NewClient(NewPoolCaller(pool))
	tx, err := c.GetTransactionFromNodes(context.Background(), "sig", CommitmentConfirmed, 2)
	if err != nil || tx == nil || tx.Slot != 7 {
		t.Fatalf("tx = %+v err = %v", tx, err)
	}
}

// NEW-M3: "absent on two nodes" reaches two distinct endpoints; a one-endpoint pool reports one.
func TestClient_GetTransactionFromDistinctNodesCountsEndpoints(t *testing.T) {
	a, aCalls := rpcServer(t, func(string, int) (int, any) { return http.StatusOK, nil })
	b, bCalls := rpcServer(t, func(string, int) (int, any) { return http.StatusOK, nil })
	pool, _ := testPool(t, a.URL, b.URL)
	c := NewClient(NewPoolCaller(pool))
	tx, reached, err := c.GetTransactionFromDistinctNodes(context.Background(), "sig", CommitmentFinalized, 2)
	if err != nil || tx != nil || reached != 2 {
		t.Fatalf("tx=%v reached=%d err=%v", tx, reached, err)
	}
	if atomic.LoadInt32(aCalls) != 1 || atomic.LoadInt32(bCalls) != 1 {
		t.Fatalf("each endpoint once: a=%d b=%d", *aCalls, *bCalls)
	}
	single, _ := testPool(t, a.URL)
	if _, reached, _ := NewClient(NewPoolCaller(single)).GetTransactionFromDistinctNodes(context.Background(), "sig", CommitmentFinalized, 2); reached != 1 {
		t.Fatalf("single pool reached = %d", reached)
	}
	if _, reached, _ := NewClient(&StaticCaller{URL: a.URL}).GetTransactionFromDistinctNodes(context.Background(), "sig", CommitmentFinalized, 2); reached != 1 {
		t.Fatalf("static caller reached = %d", reached)
	}
}

// NEW-L3: with every node backing off, Call returns ErrRateLimited at once instead of sleeping.
func TestPoolCaller_DoesNotSleepWhenAllNodesBackOff(t *testing.T) {
	limited, _ := rpcServer(t, func(string, int) (int, any) { return http.StatusTooManyRequests, nil })
	pool, _ := testPool(t, limited.URL)
	caller := NewPoolCaller(pool)
	caller.backoffBase = 10 * time.Second
	var out uint64
	start := time.Now()
	_ = caller.Call(context.Background(), "getSlot", nil, &out)
	err := caller.Call(context.Background(), "getSlot", nil, &out)
	if !errors.Is(err, ErrRateLimited) || time.Since(start) > time.Second {
		t.Fatalf("err=%v took %s", err, time.Since(start))
	}
}
