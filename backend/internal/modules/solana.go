package modules

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// SolanaModule is the wired Solana chain: the adapter the registry registers, the RPC client the
// watcher and sweeper share, the chain row, and the sweep identity. The watcher and sweeper
// services themselves live in internal/service because they compose DepositService, KeyResolver
// and SweepService, which this package must not import.
type SolanaModule struct {
	Adapter *solana.Adapter
	Client  *solana.Client
	Chain   *models.Blockchain
	Cluster string
	// FeePayer and HotWallet are set only when sweeping is configured.
	FeePayer  *solana.Ed25519Signer
	HotWallet solana.PublicKey
	// LateWindow is how long after payment expiry an account stays watched.
	LateWindow time.Duration
	// Tokens are the enabled SPL rows after validation against the chain.
	Tokens []models.BlockchainCurrency
}

// ErrSolanaNotSeeded means the SOLANA chain row is absent; the module is simply not wired.
var ErrSolanaNotSeeded = errors.New("solana: chain SOLANA is not seeded")

// publicEndpoints are Solana Labs' shared, rate limited RPC hosts; live needs a provider.
var publicEndpoints = []string{"api.mainnet-beta.solana.com", "api.devnet.solana.com", "api.testnet.solana.com"}

// IsPublicSolanaEndpoint reports a Solana Labs public RPC host.
func IsPublicSolanaEndpoint(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, p := range publicEndpoints {
		if host == p {
			return true
		}
	}
	return false
}

// SolanaCluster derives the cluster from SOLANA_CLUSTER or the network mode.
func SolanaCluster(cfg *config.Config) string {
	if c := strings.TrimSpace(cfg.Solana.Cluster); c != "" {
		return c
	}
	if cfg.Blockchain.NetworkType == "mainnet" {
		return solana.ClusterMainnet
	}
	return solana.ClusterDevnet
}

// WireSolana loads the SOLANA chain and its RPC nodes, refuses the public endpoint in live, parses
// the sweep identity without ever logging key material, validates every enabled mint against the
// chain and fills the devnet USDT row from SOLANA_DEVNET_USDT_MINT.
func WireSolana(deps Deps) (*SolanaModule, error) {
	if deps.DB == nil || deps.Config == nil {
		return nil, fmt.Errorf("modules: solana needs DB and Config")
	}
	cfg := deps.Config
	env, err := environment.Parse(cfg.Gateway.Environment)
	if err != nil {
		return nil, err
	}
	chainRepo := repository.NewBlockchainRepository(deps.DB)
	chain, err := chainRepo.GetByCode(solana.ChainCode)
	if err != nil {
		return nil, ErrSolanaNotSeeded
	}
	if chain.Status != "active" {
		return nil, fmt.Errorf("%w: status %s", ErrSolanaNotSeeded, chain.Status)
	}
	nodeRepo := repository.NewRPCNodeRepository(deps.DB)
	pool := blockchain.NewRPCPool(chain.ID, nodeRepo)
	if err := pool.Refresh(); err != nil {
		return nil, fmt.Errorf("solana: rpc nodes: %w", err)
	}
	if pool.Len() == 0 {
		return nil, fmt.Errorf("solana: no healthy rpc_nodes rows for chain %s", chain.Code)
	}
	nodes, err := nodeRepo.ListHealthyByBlockchain(chain.ID)
	if err != nil {
		return nil, err
	}
	if err := requireProviderEndpoint(env, nodes); err != nil {
		return nil, err
	}
	cluster := SolanaCluster(cfg)
	if env == environment.Live && cluster != solana.ClusterMainnet {
		return nil, fmt.Errorf("%w: live money requires cluster %s, got %s", environment.ErrBoot, solana.ClusterMainnet, cluster)
	}
	caller := solana.NewPoolCaller(pool).WithRateLimit(float64(cfg.Solana.RequestsPerSecond))
	adapter := solana.NewAdapterWithCaller(caller, cluster)
	m := &SolanaModule{
		Adapter: adapter, Client: adapter.Client(), Chain: chain, Cluster: cluster,
		LateWindow: time.Duration(max(cfg.Solana.LateWindowDays, 1)) * 24 * time.Hour,
	}

	if cfg.Solana.HotWalletAddress != "" || cfg.Solana.FeePayerKey != "" {
		if cfg.Solana.HotWalletAddress == "" || cfg.Solana.FeePayerKey == "" {
			return nil, errors.New("solana: SOLANA_HOT_WALLET_ADDRESS and SOLANA_FEE_PAYER_KEY must be set together")
		}
		hot, err := solana.ParsePublicKey(cfg.Solana.HotWalletAddress)
		if err != nil {
			return nil, errors.New("solana: SOLANA_HOT_WALLET_ADDRESS is not a public key")
		}
		feePayer, err := solana.ParseKeypair(cfg.Solana.FeePayerKey)
		if err != nil {
			// The parser's errors never carry the input; this message adds nothing either.
			return nil, fmt.Errorf("solana: SOLANA_FEE_PAYER_KEY: %w", err)
		}
		m.FeePayer, m.HotWallet = &feePayer, hot
	}

	currencies := repository.NewBlockchainCurrencyRepository(deps.DB)
	if cluster != solana.ClusterMainnet {
		applyDevnetUSDTMint(currencies, cfg.Solana.DevnetUSDTMint)
	}
	rows, err := currencies.ListByBlockchainID(chain.ID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.Address == "" || !row.DepositEnabled {
			continue
		}
		if err := validateMint(m.Client, row); err != nil {
			if env == environment.Live {
				return nil, fmt.Errorf("%w: %v", environment.ErrBoot, err)
			}
			log.Printf("[solana] MINT MISMATCH, deposits for %s disabled until the seed is fixed: %v", row.CurrencyCode, err)
			row.DepositEnabled = false
			if uerr := currencies.Update(&row); uerr != nil {
				log.Printf("[solana] disable %s deposits: %v", row.CurrencyCode, uerr)
			}
			continue
		}
		m.Tokens = append(m.Tokens, row)
	}
	return m, nil
}

// requireProviderEndpoint refuses, in live, a pool made only of Solana Labs' public endpoints, and a
// pool with fewer than two distinct endpoints: drop and expiry evidence needs two (sweeps cannot
// rebuild without it; see service/SOLANA_SWEEPS.md).
func requireProviderEndpoint(env environment.Environment, nodes []models.RPCNode) error {
	if env != environment.Live {
		return nil
	}
	distinct := map[string]bool{}
	provider := false
	for _, n := range nodes {
		distinct[normalizeEndpoint(n.URL)] = true
		provider = provider || !IsPublicSolanaEndpoint(n.URL)
	}
	if !provider {
		return fmt.Errorf("%w: solana rpc_nodes hold only public endpoints; live needs a provider endpoint (add an rpc_nodes row)", environment.ErrBoot)
	}
	if len(distinct) < 2 {
		return fmt.Errorf("%w: solana rpc_nodes hold %d distinct endpoint; live needs at least two for drop and expiry evidence (add an rpc_nodes row)", environment.ErrBoot, len(distinct))
	}
	return nil
}

// normalizeEndpoint compares endpoint URLs case-insensitively and without a trailing slash.
func normalizeEndpoint(raw string) string {
	return strings.TrimRight(strings.ToLower(strings.TrimSpace(raw)), "/")
}

// validateMint checks the seeded token program and decimals against the mint account on chain.
func validateMint(c *solana.Client, row models.BlockchainCurrency) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	info, err := c.GetMint(ctx, row.Address)
	if err != nil {
		return fmt.Errorf("mint %s for %s: %w", row.Address, row.CurrencyCode, err)
	}
	program, err := solana.TokenProgramFor(row.Standard)
	if err != nil {
		return fmt.Errorf("%s: %w", row.CurrencyCode, err)
	}
	if info.TokenProgram != program {
		return fmt.Errorf("%s: seeded standard %s expects program %s but the mint is owned by %s", row.CurrencyCode, row.Standard, program, info.TokenProgram)
	}
	if uint(info.Decimals) != row.WalletPrecision {
		return fmt.Errorf("%s: seeded wallet_precision %d but the mint has %d decimals", row.CurrencyCode, row.WalletPrecision, info.Decimals)
	}
	return nil
}

// applyDevnetUSDTMint fills the seeded devnet USDT row, which ships disabled with no mint because
// devnet has no official USDT. Logged either way so the state is never silent.
func applyDevnetUSDTMint(currencies repository.BlockchainCurrencyRepository, mint string) {
	row, err := currencies.GetByBlockchainCodeAndCurrencyCode(solana.ChainCode, "USDT")
	if err != nil {
		return
	}
	if mint == "" {
		if row.Address == "" {
			log.Printf("[solana] devnet USDT has no mint; set SOLANA_DEVNET_USDT_MINT to enable it")
		}
		return
	}
	if _, err := solana.ParsePublicKey(mint); err != nil {
		log.Printf("[solana] SOLANA_DEVNET_USDT_MINT ignored: not a public key")
		return
	}
	if row.Address == mint && row.DepositEnabled {
		return
	}
	row.Address, row.DepositEnabled, row.Visible = mint, true, true
	if err := currencies.Update(row); err != nil {
		log.Printf("[solana] devnet USDT mint update: %v", err)
		return
	}
	log.Printf("[solana] devnet USDT mint set to %s", mint)
}
