package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/gorm"
)

func main() {
	var (
		network  = flag.String("network", "", "Network mode for seed selection: testnet|mainnet (default: $BLOCKCHAIN_NETWORK_TYPE)")
		seedDir  = flag.String("seeds", "migrations/seeds", "Directory holding seed SQL files")
		seed     = flag.Bool("seed", false, "Explicitly apply network catalog seeds after migrations")
		seedOnly = flag.Bool("seed-only", false, "Deprecated alias for the explicit seed action")
		appRole  = flag.String("ledger-app-role", "", "Role to narrow to SELECT, INSERT on the ledger tables (default: $POSTGRES_LEDGER_APP_ROLE)")
		adopt    = flag.String("confirm-adopt-live", "", "adopt-live only: the name of the connected database, typed out as confirmation")
		adoptT   = flag.String("confirm-adopt-test", "", "adopt-test only: the name of the connected database, typed out as confirmation")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	mode := cmp.Or(*network, cfg.Blockchain.NetworkType, "testnet")
	if mode != "testnet" && mode != "mainnet" {
		log.Fatalf("invalid --network %q, must be testnet or mainnet", mode)
	}

	envModule, err := modules.WireEnvironment(modules.Deps{Config: cfg})
	if err != nil {
		log.Fatalf("environment: %v", err)
	}

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	action := "up"
	if flag.NArg() > 0 {
		action = flag.Arg(0)
	}
	// The adoption actions are the ones that expect an unstamped or disagreeing database.
	if action != "adopt-live" && action != "adopt-test" {
		if err := envModule.VerifyDatabase(context.Background(), db); err != nil {
			log.Fatalf("environment: %v", err)
		}
	}

	switch action {
	case "adopt-live":
		result, err := envModule.AdoptLive(context.Background(), db, *adopt)
		if err != nil {
			log.Fatalf("adopt-live: %v", err)
		}
		log.Printf("adopt-live: %s is now live (ledger accounts %d, journals %d, legacy api keys %d)", result.Database, result.LedgerAccounts, result.LedgerJournals, result.APIKeys)
	case "adopt-test":
		name, err := envModule.AdoptTest(context.Background(), db, *adoptT)
		if err != nil {
			log.Fatalf("adopt-test: %v", err)
		}
		log.Printf("adopt-test: %s is stamped test", name)
	case "up":
		if !*seedOnly {
			log.Println("Applying checksummed schema migrations...")
			results, err := database.ApplyMigrations(context.Background(), db)
			if err != nil {
				log.Fatalf("schema migrate: %v", err)
			}
			for _, result := range results {
				log.Printf("  applied %d %s (%s)", result.Version, result.Name, result.Checksum)
			}
			if err := envModule.VerifySchema(context.Background(), db); err != nil {
				log.Fatalf("environment: %v", err)
			}
			modeRepo := service.NewConfigRepoAdapter(repository.NewConfigurationRepository(db))
			if err := config.CheckModeMatch(cfg.Blockchain.NetworkType, modeRepo); err != nil {
				log.Fatalf("mode enforcement: %v", err)
			}
			if err := envModule.Finalize(context.Background(), db, cfg.Blockchain.NetworkType); err != nil {
				log.Fatalf("environment: %v", err)
			}
			log.Printf("  environment: database stamped %s", envModule.Environment)
			if role := cmp.Or(*appRole, cfg.Database.LedgerAppRole); role != "" {
				if err := ledger.GrantAppRole(db, role); err != nil {
					log.Fatalf("ledger app role: %v", err)
				}
				log.Printf("  ledger: role %s limited to SELECT, INSERT", role)
			}
		}
		if *seed || *seedOnly {
			log.Printf("Applying explicitly requested %s seeds from %s/%s ...", mode, *seedDir, mode)
			if err := runSeeds(db, filepath.Join(*seedDir, mode)); err != nil {
				log.Fatalf("seed: %v", err)
			}
		}
		log.Println("Migration complete.")
	case "seed":
		log.Printf("Applying explicitly requested %s seeds from %s/%s ...", mode, *seedDir, mode)
		if err := runSeeds(db, filepath.Join(*seedDir, mode)); err != nil {
			log.Fatalf("seed: %v", err)
		}
	case "down":
		log.Fatal("down: unsupported; migrations are forward-only and require a reviewed roll-forward plan")
	default:
		log.Fatalf("unknown action %q (try: up, seed, adopt-live, adopt-test)", action)
	}
}

// runSeeds reads every *.sql file in dir, sorts them by filename, and
// executes each in its own transaction. ON CONFLICT clauses make it safe
// to re-run.
func runSeeds(db *gorm.DB, dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("seeds directory %q not found — skipping", dir)
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %q: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	slices.Sort(files)

	for _, name := range files {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		log.Printf("  applying %s", name)
		if err := db.Transaction(func(tx *gorm.DB) error {
			return tx.Exec(string(raw)).Error
		}); err != nil {
			return fmt.Errorf("execute %s: %w", path, err)
		}
	}
	return nil
}
