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

// Anomaly is an asset the checkpoint could not state honestly, with the reason; it is logged and omitted.
type Anomaly struct {
	Asset  string
	Reason string
}

func (a Anomaly) String() string { return a.Asset + ": " + a.Reason }

// BuildCheckpoint reads liabilities through the ledger port and hashes them so a solvency report can name them.
// Assets with unknown decimals or a negative total are left out and returned as anomalies; nothing is guessed.
func BuildCheckpoint(ctx context.Context, src LiabilitySource, decimals Decimals, now time.Time) (Checkpoint, []Anomaly, error) {
	snap, err := src.LiabilityTotals(ctx)
	if err != nil {
		return Checkpoint{}, nil, err
	}
	totals, maxJournal, takenAt := snap.Totals, snap.Head, snap.TakenAt
	if takenAt.IsZero() {
		takenAt = now
	}
	var skipped []Anomaly
	assets := make([]AssetTotal, 0, len(totals))
	for _, t := range totals {
		minor, dec, ok := decimals.Minor(t.Asset, t.Total)
		if !ok {
			skipped = append(skipped, Anomaly{Asset: t.Asset, Reason: "unknown decimals or amount off the grid (set CRE_ASSET_DECIMALS)"})
			continue
		}
		if minor.Sign() < 0 {
			skipped = append(skipped, Anomaly{Asset: t.Asset, Reason: "negative liability total " + minor.String() + "; a ledger defect, not attested"})
			continue
		}
		assets = append(assets, AssetTotal{Asset: t.Asset, Liabilities: minor, Decimals: dec})
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Asset < assets[j].Asset })
	cp := Checkpoint{ID: uuid.NewString(), TakenAt: takenAt.UTC(), MaxJournalID: maxJournal, Assets: assets}
	cp.Hash = CheckpointHash(cp)
	return cp, skipped, nil
}

// CheckpointTime is the taken_at string the hash commits to and the checkpoint facts store.
func CheckpointTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

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
	}{MaxJournalID: cp.MaxJournalID, TakenAt: CheckpointTime(cp.TakenAt), Assets: make([]asset, 0, len(cp.Assets))}
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
