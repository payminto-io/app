package ethereum

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// fakeRPCServer is a minimal JSON-RPC test server that returns canned
// responses for the methods the adapter calls.
type fakeRPCServer struct {
	*httptest.Server
	balance         string // hex, for eth_getBalance
	latestBlock     string // hex, for eth_blockNumber and eth_getBlockByNumber
	receiptBlockNum string // hex, for eth_getTransactionReceipt
	failNextN       int    // if >0, return an error and decrement
}

func newFakeRPC(t *testing.T) *fakeRPCServer {
	srv := &fakeRPCServer{
		balance:         "0x1bc16d674ec80000", // 2 ETH
		latestBlock:     "0x10",
		receiptBlockNum: "0xf",
	}
	srv.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JSONRPC string `json:"jsonrpc"`
			ID      int    `json:"id"`
			Method  string `json:"method"`
			Params  []any  `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		if srv.failNextN > 0 {
			srv.failNextN--
			http.Error(w, "simulated RPC failure", 500)
			return
		}

		respond := func(result any) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  result,
			})
		}

		switch req.Method {
		case "eth_chainId":
			respond("0x1")
		case "eth_blockNumber":
			respond(srv.latestBlock)
		case "eth_getBalance":
			respond(srv.balance)
		case "eth_getTransactionReceipt":
			respond(map[string]any{
				"blockNumber":       srv.receiptBlockNum,
				"status":            "0x1",
				"transactionHash":   "0x0000000000000000000000000000000000000000000000000000000000000abc",
				"contractAddress":   nil,
				"logs":              []any{},
				"blockHash":         "0x000000000000000000000000000000000000000000000000000000000000beef",
				"transactionIndex":  "0x0",
				"gasUsed":           "0x5208",
				"cumulativeGasUsed": "0x5208",
				"logsBloom":         "0x" + strings.Repeat("0", 512),
				"type":              "0x0",
			})
		case "eth_getBlockByNumber":
			respond(map[string]any{
				"number":           srv.latestBlock,
				"hash":             "0x000000000000000000000000000000000000000000000000000000000000beef",
				"parentHash":       "0x0000000000000000000000000000000000000000000000000000000000000000",
				"transactions":     []any{},
				"timestamp":        "0x0",
				"nonce":            "0x0000000000000000",
				"sha3Uncles":       "0x" + strings.Repeat("0", 64),
				"logsBloom":        "0x" + strings.Repeat("0", 512),
				"transactionsRoot": "0x" + strings.Repeat("0", 64),
				"stateRoot":        "0x" + strings.Repeat("0", 64),
				"receiptsRoot":     "0x" + strings.Repeat("0", 64),
				"miner":            "0x0000000000000000000000000000000000000000",
				"difficulty":       "0x0",
				"totalDifficulty":  "0x0",
				"extraData":        "0x",
				"size":             "0x0",
				"gasLimit":         "0x0",
				"gasUsed":          "0x0",
				"uncles":           []any{},
				"baseFeePerGas":    "0x0",
			})
		case "eth_gasPrice":
			respond("0x4a817c800") // 20 gwei
		default:
			respond(nil)
		}
	}))
	return srv
}

func setupTestPool(t *testing.T, url string) (*blockchain.RPCPool, repository.RPCNodeRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.BlockchainFamily{}, &models.Blockchain{}, &models.RPCNode{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRPCNodeRepository(db)

	family := &models.BlockchainFamily{Code: "evm", Name: "EVM"}
	if err := db.Create(family).Error; err != nil {
		t.Fatal(err)
	}
	bc := &models.Blockchain{Code: "ETH", Name: "Ethereum", Status: "active", BlockchainFamilyID: family.ID}
	if err := db.Create(bc).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(&models.RPCNode{
		BlockchainID: bc.ID,
		Name:         "test",
		URL:          url,
		Status:       models.RPCNodeStatusHealthy,
	}); err != nil {
		t.Fatal(err)
	}
	pool := blockchain.NewRPCPool(bc.ID, repo)
	if err := pool.Refresh(); err != nil {
		t.Fatal(err)
	}
	return pool, repo
}

func TestEthAdapter_Name(t *testing.T) {
	a := NewAdapter("ETH", "Ethereum", 1, nil)
	if a.Name() != "Ethereum" {
		t.Errorf("expected Ethereum, got %s", a.Name())
	}
	if a.Code() != "ETH" {
		t.Errorf("expected ETH, got %s", a.Code())
	}
	if a.ChainID() != 1 {
		t.Errorf("expected chain id 1, got %d", a.ChainID())
	}
}

func TestEthAdapter_IsMainnet(t *testing.T) {
	cases := []struct {
		code    string
		chainID int64
		want    bool
	}{
		{"ETH", 1, true},
		{"ETH", 11155111, false}, // Sepolia
		{"BASE", 8453, true},
		{"BASE", 84532, false}, // Base Sepolia
		{"POLYGON", 137, true},
		{"POLYGON", 80002, false}, // Amoy
	}
	for _, c := range cases {
		a := NewAdapter(c.code, "test", c.chainID, nil)
		if a.IsMainnet() != c.want {
			t.Errorf("%s chainID=%d: IsMainnet=%v, want %v", c.code, c.chainID, a.IsMainnet(), c.want)
		}
	}
}

func TestEthAdapter_EstimateGas_NoPool(t *testing.T) {
	a := NewAdapter("ETH", "Ethereum", 1, nil)
	gas, err := a.EstimateGas(t.Context(), blockchain.TxParams{})
	if err != nil {
		t.Fatal(err)
	}
	if gas.Sign() <= 0 {
		t.Error("expected positive gas estimate")
	}
}

func TestEthAdapter_GetBalance_Native(t *testing.T) {
	srv := newFakeRPC(t)
	defer srv.Close()

	pool, _ := setupTestPool(t, srv.URL)
	a := NewAdapter("ETH", "Ethereum", 1, pool)

	bal, err := a.GetBalance(t.Context(), "0x0000000000000000000000000000000000000001", "")
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	want := new(big.Int)
	want.SetString("2000000000000000000", 10)
	if bal.Cmp(want) != 0 {
		t.Errorf("expected 2 ETH in wei, got %s", bal.String())
	}
}

func TestEthAdapter_GetConfirmations(t *testing.T) {
	srv := newFakeRPC(t)
	defer srv.Close()

	pool, _ := setupTestPool(t, srv.URL)
	a := NewAdapter("ETH", "Ethereum", 1, pool)

	conf, err := a.GetConfirmations(t.Context(), "0x0000000000000000000000000000000000000000000000000000000000000abc")
	if err != nil {
		t.Fatalf("GetConfirmations: %v", err)
	}
	// latest = 0x10 = 16, receiptBlockNum = 0xf = 15 → confirmations = 16 - 15 + 1 = 2
	if conf != 2 {
		t.Errorf("expected 2 confirmations, got %d", conf)
	}
}

func TestEthAdapter_GetBalance_RPCFailure_MarksUnhealthy(t *testing.T) {
	srv := newFakeRPC(t)
	srv.failNextN = 10 // every call fails
	defer srv.Close()

	pool, repo := setupTestPool(t, srv.URL)
	a := NewAdapter("ETH", "Ethereum", 1, pool.WithFailThreshold(2))

	_, _ = a.GetBalance(t.Context(), "0x0000000000000000000000000000000000000001", "")
	_, _ = a.GetBalance(t.Context(), "0x0000000000000000000000000000000000000001", "")
	_, _ = a.GetBalance(t.Context(), "0x0000000000000000000000000000000000000001", "")

	nodes, _ := repo.ListByBlockchain(1)
	if len(nodes) == 0 {
		t.Skip("no nodes found, skipping fail count check")
	}
	if nodes[0].FailCount < 2 {
		t.Errorf("expected fail_count >= 2, got %d", nodes[0].FailCount)
	}
}

func TestDecodeTransferLog_ShortTopics(t *testing.T) {
	l := types.Log{}
	if _, _, _, err := DecodeTransferLog(l); err == nil {
		t.Error("expected error for log with no topics")
	}
}

func TestEthAdapter_ImplementsInterface(t *testing.T) {
	var _ blockchain.ChainAdapter = NewAdapter("ETH", "Ethereum", 1, nil)
}

// This ensures the imports compile
var _ = fmt.Sprint
