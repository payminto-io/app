package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/blockchain/tempo"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

func TestTempoWiringRejectsWrongChain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID json.RawMessage `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": "0x1"})
	}))
	defer server.Close()
	svc, db, _ := newTempoFixture(t, &fakeTempoReader{})
	r := &ServiceRegistry{db: db, blockchainRepo: repository.NewBlockchainRepository(db), blockchainCurrencyRepo: repository.NewBlockchainCurrencyRepository(db), depositService: svc.depositSvc}
	cfg := &config.Config{Gateway: config.GatewayConfig{Environment: "test"}, Blockchain: config.BlockchainConfig{NetworkType: "testnet"},
		Tempo: config.TempoConfig{MerchantAddress: tempoMerchant.Hex(), RPCURL: server.URL, PollIntervalSeconds: 2}}
	if err := r.wireTempo(cfg); err == nil {
		t.Fatal("wrong RPC chain accepted by wiring")
	}
	if r.tempoMemoService != nil {
		t.Fatal("watcher wired despite chain mismatch")
	}
}

func TestTempoAssignmentRejectsDisabledAndOtherMerchant(t *testing.T) {
	for _, kind := range []string{"merchant", "disabled token", "inactive chain"} {
		t.Run(kind, func(t *testing.T) {
			_, db, p := newTempoFixture(t, &fakeTempoReader{})
			addressSvc := NewDepositAddressService(repository.NewDepositAddressRepository(db), nil, repository.NewBlockchainCurrencyRepository(db), nil, nil).
				WithTempoMerchant(tempoMerchant.Hex(), p.MemberID)
			switch kind {
			case "merchant":
				addressSvc.WithTempoMerchant(tempoMerchant.Hex(), p.MemberID+1)
			case "disabled token":
				if err := db.Model(&models.BlockchainCurrency{}).Where("blockchain_code = ?", "TEMPO").Update("deposit_enabled", false).Error; err != nil {
					t.Fatal(err)
				}
			case "inactive chain":
				if err := db.Model(&models.Blockchain{}).Where("code = ?", "TEMPO").Update("status", "inactive").Error; err != nil {
					t.Fatal(err)
				}
			}
			if _, err := addressSvc.AssignForPayment(p, "TEMPO", "ALPHAUSD"); err == nil {
				t.Fatal("unavailable Tempo leg assigned")
			}
		})
	}
}

func TestTempoMemoRecipientAndAmountRejections(t *testing.T) {
	reader := &fakeTempoReader{}
	svc, db, p := newTempoFixture(t, reader)
	for i, tr := range []struct {
		recipient string
		amount    int64
	}{{"0x0000000000000000000000000000000000000001", 2_000_000}, {tempoMerchant.Hex(), 0}} {
		transfer := memoTransfer(p.ReferenceID, tempo.AlphaUSD, common.HexToAddress(tr.recipient), tr.amount, "0xf1", uint(i), 110)
		if credited, err := svc.handle(context.Background(), transfer); err != nil || credited {
			t.Fatalf("rejected transfer credited=%t, err=%v", credited, err)
		}
	}
	var count int64
	if err := db.Model(&models.Deposit{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("deposits=%d, err=%v", count, err)
	}
}
