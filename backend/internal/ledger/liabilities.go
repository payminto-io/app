package ledger

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

// LiabilityTotal is the natural (positive) balance owed to members per asset.
type LiabilityTotal struct {
	Asset string
	Total decimal.Decimal
}

// LiabilityTotals sums every member liability account per asset in the process environment and
// returns the newest journal id those sums include; the attestation module hashes both (ticket 21).
func (s *Service) LiabilityTotals(ctx context.Context) ([]LiabilityTotal, uint64, error) {
	env, err := s.resolve(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	db := s.db.WithContext(ctx)
	var rows []struct {
		Asset string
		Total decimal.Decimal
	}
	err = db.Model(&AccountRow{}).
		Select("ledger_accounts.asset, COALESCE(SUM(ledger_lines.amount), 0) AS total").
		Joins("LEFT JOIN ledger_lines ON ledger_lines.account_id = ledger_accounts.id").
		Where("ledger_accounts.environment = ? AND ledger_accounts.owner_type = ? AND ledger_accounts.kind = ?", env, OwnerMember, KindLiability).
		Group("ledger_accounts.asset").
		Order("ledger_accounts.asset").
		Scan(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("ledger: liability totals: %w", err)
	}
	var head struct{ Max uint64 }
	if err := db.Model(&JournalRow{}).Select("COALESCE(MAX(id), 0) AS max").Where("environment = ?", env).Scan(&head).Error; err != nil {
		return nil, 0, fmt.Errorf("ledger: journal head: %w", err)
	}
	out := make([]LiabilityTotal, 0, len(rows))
	for _, r := range rows {
		out = append(out, LiabilityTotal{Asset: r.Asset, Total: NaturalBalance(KindLiability, r.Total)})
	}
	return out, head.Max, nil
}
