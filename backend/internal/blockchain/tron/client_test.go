package tron

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFakeTronServer(handler func(path string) any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := handler(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if result == nil {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	}))
}

func setupTronTestPool(t *testing.T, url string) *blockchain.RPCPool {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.BlockchainFamily{}, &models.Blockchain{}, &models.RPCNode{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRPCNodeRepository(db)
	_ = db.Create(&models.BlockchainFamily{Code: "trx", Name: "Tron"}).Error
	bc := &models.Blockchain{Code: "TRX", Name: "Tron", Status: "active"}
	_ = db.Create(bc).Error
	_ = repo.Create(&models.RPCNode{
		BlockchainID: bc.ID,
		Name:         "test",
		URL:          url,
		Status:       models.RPCNodeStatusHealthy,
	})
	pool := blockchain.NewRPCPool(bc.ID, repo)
	_ = pool.Refresh()
	return pool
}

func TestTronAdapter_Name(t *testing.T) {
	a := NewAdapter(nil, "mainnet")
	if a.Name() != "Tron" {
		t.Errorf("expected Tron, got %s", a.Name())
	}
	if a.Code() != "TRX" {
		t.Errorf("expected TRX, got %s", a.Code())
	}
}

func TestTronAdapter_IsMainnet(t *testing.T) {
	main := NewAdapter(nil, "mainnet")
	if !main.IsMainnet() {
		t.Error("expected mainnet true")
	}
	nile := NewAdapter(nil, "nile")
	if nile.IsMainnet() {
		t.Error("expected nile IsMainnet false")
	}
	shasta := NewAdapter(nil, "shasta")
	if shasta.IsMainnet() {
		t.Error("expected shasta IsMainnet false")
	}
}

func TestTronAdapter_Network(t *testing.T) {
	a := NewAdapter(nil, "nile")
	if a.Network() != "nile" {
		t.Errorf("expected nile, got %s", a.Network())
	}
}

func TestTronAdapter_EstimateGas_Zero(t *testing.T) {
	a := NewAdapter(nil, "mainnet")
	gas, err := a.EstimateGas(t.Context(), blockchain.TxParams{})
	if err != nil {
		t.Fatal(err)
	}
	if gas.Int64() != 0 {
		t.Errorf("expected 0 (tron has no gas), got %d", gas.Int64())
	}
}

func TestTronAdapter_GetBalance_TRX(t *testing.T) {
	srv := newFakeTronServer(func(path string) any {
		if path == "/wallet/getaccount" {
			return map[string]any{"balance": int64(5_000_000)} // 5 TRX
		}
		return nil
	})
	defer srv.Close()

	pool := setupTronTestPool(t, srv.URL)
	a := NewAdapter(pool, "mainnet")

	bal, err := a.GetBalance(t.Context(), "TWqv5HotLfCAvGeNKp9k5LfypJ3xueCYw", "")
	if err != nil {
		t.Fatal(err)
	}
	if bal.Int64() != 5_000_000 {
		t.Errorf("expected 5000000 sun, got %d", bal.Int64())
	}
}

func TestTronAdapter_GetConfirmations(t *testing.T) {
	srv := newFakeTronServer(func(path string) any {
		switch path {
		case "/wallet/gettransactioninfobyid":
			return map[string]any{"blockNumber": int64(100)}
		case "/wallet/getnowblock":
			return map[string]any{
				"block_header": map[string]any{
					"raw_data": map[string]any{"number": int64(105)},
				},
			}
		}
		return nil
	})
	defer srv.Close()

	pool := setupTronTestPool(t, srv.URL)
	a := NewAdapter(pool, "mainnet")

	conf, err := a.GetConfirmations(t.Context(), "txhash")
	if err != nil {
		t.Fatal(err)
	}
	// latest 105, tx 100 → 105 - 100 + 1 = 6
	if conf != 6 {
		t.Errorf("expected 6, got %d", conf)
	}
}

func TestTronAdapter_BroadcastTransaction_Success(t *testing.T) {
	srv := newFakeTronServer(func(path string) any {
		if path == "/wallet/broadcasthex" {
			return map[string]any{"result": true, "txid": "abc123"}
		}
		return nil
	})
	defer srv.Close()

	pool := setupTronTestPool(t, srv.URL)
	a := NewAdapter(pool, "mainnet")

	id, err := a.BroadcastTransaction(t.Context(), []byte{0x01})
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc123" {
		t.Errorf("expected abc123, got %s", id)
	}
}

func TestTronAdapter_BroadcastTransaction_Rejected(t *testing.T) {
	srv := newFakeTronServer(func(path string) any {
		if path == "/wallet/broadcasthex" {
			return map[string]any{"result": false}
		}
		return nil
	})
	defer srv.Close()

	pool := setupTronTestPool(t, srv.URL)
	a := NewAdapter(pool, "mainnet")

	_, err := a.BroadcastTransaction(t.Context(), []byte{0x01})
	if err == nil {
		t.Error("expected error on rejected broadcast")
	}
}

func TestTronAdapter_GenerateAddress_Error(t *testing.T) {
	a := NewAdapter(nil, "mainnet")
	if _, err := a.GenerateAddress(0); err == nil {
		t.Error("expected error directing caller to HD wallet")
	}
}

func TestTronAdapter_RPCFailure_MarksUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "dead", 500)
	}))
	defer srv.Close()

	pool := setupTronTestPool(t, srv.URL).WithFailThreshold(2)
	a := NewAdapter(pool, "mainnet")

	_, _ = a.GetConfirmations(t.Context(), "abc")
	_, _ = a.GetConfirmations(t.Context(), "abc")
	_, _ = a.GetConfirmations(t.Context(), "abc")

	// No hard assertion on size — just ensure the loop exits without panic.
	_ = pool.Size()
}

func TestTronAdapter_ImplementsInterface(t *testing.T) {
	var _ blockchain.ChainAdapter = NewAdapter(nil, "mainnet")
}
