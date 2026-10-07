package cre

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
)

// BuildCheckpoint reads liabilities through the ledger port and hashes them so a solvency report can name them.
// Assets with unknown decimals are left out and returned in skipped; nothing is guessed.
func BuildCheckpoint(ctx context.Context, src LiabilitySource, decimals Decimals, now time.Time) (Checkpoint, []string, error) {
	totals, maxJournal, err := src.LiabilityTotals(ctx)
	if err != nil {
		return Checkpoint{}, nil, err
	}
	var skipped []string
	assets := make([]AssetTotal, 0, len(totals))
	for _, t := range totals {
		minor, dec, ok := decimals.Minor(t.Asset, t.Total)
		if !ok {
			skipped = append(skipped, t.Asset)
			continue
		}
		if minor.Sign() < 0 {
			minor.SetInt64(0)
		}
		assets = append(assets, AssetTotal{Asset: t.Asset, Liabilities: minor, Decimals: dec})
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Asset < assets[j].Asset })
	cp := Checkpoint{ID: uuid.NewString(), TakenAt: now.UTC(), MaxJournalID: maxJournal, Assets: assets}
	cp.Hash = CheckpointHash(cp)
	return cp, skipped, nil
}

// CheckpointHash is keccak256 of the canonical JSON of the checkpoint's facts; an auditor can replay it.
func CheckpointHash(cp Checkpoint) [32]byte {
	type asset struct {
		Asset       string `json:"asset"`
		Liabilities string `json:"liabilities_minor"`
		Decimals    uint8  `json:"decimals"`
	}
	canon := struct {
		MaxJournalID uint64  `json:"max_journal_id"`
		TakenAt      string  `json:"taken_at"`
		Assets       []asset `json:"assets"`
	}{MaxJournalID: cp.MaxJournalID, TakenAt: cp.TakenAt.UTC().Format(time.RFC3339), Assets: make([]asset, 0, len(cp.Assets))}
	for _, a := range cp.Assets {
		canon.Assets = append(canon.Assets, asset{a.Asset, a.Liabilities.String(), a.Decimals})
	}
	raw, err := json.Marshal(canon)
	if err != nil {
		panic(fmt.Sprintf("cre: checkpoint json: %v", err))
	}
	var out [32]byte
	copy(out[:], crypto.Keccak256(raw))
	return out
}
