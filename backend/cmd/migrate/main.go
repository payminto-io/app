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
	"gorm.io/gorm"
)

func main() {
	var (
		network  = flag.String("network", "", "Network mode for seed selection: testnet|mainnet (default: $BLOCKCHAIN_NETWORK_TYPE)")
		seedDir  = flag.String("seeds", "migrations/seeds", "Directory holding seed SQL files")
		seed     = flag.Bool("seed", false, "Explicitly apply network catalog seeds after migrations")
		seedOnly = flag.Bool("seed-only", false, "Deprecated alias for the explicit seed action")
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

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	action := "up"
	if flag.NArg() > 0 {
		action = flag.Arg(0)
	}

	switch action {
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
		log.Fatalf("unknown action %q (try: up, down)", action)
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
