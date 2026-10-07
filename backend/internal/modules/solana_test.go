package modules

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func solanaTestDB(t *testing.T, nodeURL string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.BlockchainFamily{}, &models.Blockchain{}, &models.RPCNode{}, &models.Currency{}, &models.BlockchainCurrency{}); err != nil {
		t.Fatal(err)
	}
	fam := models.BlockchainFamily{Code: "sol", Name: "Solana", Family: "SOL_Family"}
	db.Create(&fam)
	chain := models.Blockchain{Code: solana.ChainCode, Name: "Solana", Family: "SOL_Family", BlockchainFamilyID: fam.ID, Status: "active", MinConfirmations: 32}
	db.Create(&chain)
	db.Create(&models.RPCNode{BlockchainID: chain.ID, Name: "n", URL: nodeURL, Status: models.RPCNodeStatusHealthy})
	return db
}

func solanaConfig(env string) *config.Config {
	cfg := &config.Config{}
	cfg.Gateway.Environment = env
	cfg.Blockchain.NetworkType = "testnet"
	cfg.Solana.RequestsPerSecond = 10
	cfg.Solana.LateWindowDays = 7
	cfg.Solana.PostDepositJournals = true
	if env == "live" {
		cfg.Blockchain.NetworkType = "mainnet"
	}
	return cfg
}

// I3: a mistyped fee payer key must fail without the key reaching the error or the log.
func TestWireSolana_MistypedFeePayerKeyNeverReachesLogs(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	good := base58.Encode(priv)
	typo := good[:len(good)-1] + "typo"
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	cfg := solanaConfig("test")
	cfg.Solana.HotWalletAddress = "HAgk14JpMQLgt6rVgv7cBQFJWFto5Dqxi472uT3DKpqk"
	cfg.Solana.FeePayerKey = typo
	_, err := WireSolana(Deps{DB: solanaTestDB(t, "http://127.0.0.1:1"), Config: cfg})
	if err == nil {
		t.Fatal("typo accepted")
	}
	log.Printf("[registry] solana: %v", err)
	for _, needle := range []string{typo, good[:16]} {
		if strings.Contains(err.Error(), needle) || strings.Contains(buf.String(), needle) {
			t.Fatalf("SECRET ECHOED: err=%q log=%q", err, buf.String())
		}
	}
}

// Live refuses a pool made only of the public endpoint.
func TestWireSolana_LiveRefusesPublicEndpoint(t *testing.T) {
	_, err := WireSolana(Deps{DB: solanaTestDB(t, "https://api.mainnet-beta.solana.com"), Config: solanaConfig("live")})
	if !errors.Is(err, environment.ErrBoot) || !strings.Contains(err.Error(), "provider endpoint") {
		t.Fatalf("err = %v", err)
	}
	if !IsPublicSolanaEndpoint("https://api.devnet.solana.com/") || IsPublicSolanaEndpoint("https://mainnet.helius-rpc.com/?api-key=x") {
		t.Fatal("endpoint classification wrong")
	}
}

func TestWireSolana_TestEnvironmentWiresWithPublicEndpoint(t *testing.T) {
	m, err := WireSolana(Deps{DB: solanaTestDB(t, "https://api.devnet.solana.com"), Config: solanaConfig("test")})
	if err != nil {
		t.Fatal(err)
	}
	if m.Adapter.Code() != solana.ChainCode || m.Cluster != solana.ClusterDevnet || m.FeePayer != nil || !m.PostDepositJournals {
		t.Fatalf("module = %+v", m)
	}
}

func TestWireSolana_NotSeededIsNotAnError(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	_ = db.AutoMigrate(&models.Blockchain{}, &models.RPCNode{}, &models.BlockchainCurrency{})
	if _, err := WireSolana(Deps{DB: db, Config: solanaConfig("test")}); !errors.Is(err, ErrSolanaNotSeeded) {
		t.Fatalf("err = %v", err)
	}
}
