package ledger

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// LiabilityTotal is the natural (positive) balance owed to members per asset.
type LiabilityTotal struct {
	Asset string
	Total decimal.Decimal
}

// LiabilitySnapshot is one consistent read: Head is a commit watermark (every journal with id <= Head is
// committed and counted) and TakenAt is the database time the snapshot was frozen (docs/cre/OPERATIONS.md).
type LiabilitySnapshot struct {
	Totals  []LiabilityTotal
	Head    uint64
	TakenAt time.Time
}

// postingBarrierKey is the advisory lock posters hold shared from before they allocate a journal id until commit.
const postingBarrierKey = `hashtextextended('ledger:posting_barrier:' || current_schema(), 0)`

// takePostingBarrier is called by every poster before its journal insert, inside the posting transaction.
func takePostingBarrier(tx *gorm.DB) error {
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	if err := tx.Exec(`SELECT pg_advisory_xact_lock_shared(` + postingBarrierKey + `)`).Error; err != nil {
		return fmt.Errorf("ledger: posting barrier: %w", err)
	}
	return nil
}

// LiabilityTotals sums every member liability account per asset in the process environment at a commit
// watermark: it takes the posting barrier exclusively (waiting for in-flight postings), then reads the head
// and the sums in one repeatable-read transaction, so "lines of journals <= Head" reproduces them exactly.
func (s *Service) LiabilityTotals(ctx context.Context) (LiabilitySnapshot, error) {
	env, err := s.resolve(ctx, "")
	if err != nil {
		return LiabilitySnapshot{}, err
	}
	if s.db.Dialector.Name() != "postgres" {
		return LiabilitySnapshot{}, errors.New("ledger: liability totals need PostgreSQL")
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return LiabilitySnapshot{}, fmt.Errorf("ledger: liability totals: %w", err)
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return LiabilitySnapshot{}, fmt.Errorf("ledger: liability totals: %w", err)
	}
	defer conn.Close()
	// A session lock must never outlive this call on a pooled connection: unless the unlock is confirmed,
	// the connection is discarded, which releases the lock.
	released := false
	defer func() {
		if !released {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(`+postingBarrierKey+`)`); err != nil {
		return LiabilitySnapshot{}, fmt.Errorf("ledger: liability totals: posting barrier: %w", err)
	}
	snap, readErr := readLiabilities(ctx, conn, string(env))
	var unlocked bool
	if err := conn.QueryRowContext(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock(`+postingBarrierKey+`)`).Scan(&unlocked); err == nil && unlocked {
		released = true
	}
	if readErr != nil {
		return LiabilitySnapshot{}, fmt.Errorf("ledger: liability totals: %w", readErr)
	}
	return snap, nil
}

func readLiabilities(ctx context.Context, conn *sql.Conn, env string) (LiabilitySnapshot, error) {
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return LiabilitySnapshot{}, err
	}
	defer tx.Rollback() //nolint:errcheck // read-only; a failed rollback leaves nothing behind
	var snap LiabilitySnapshot
	// The first statement freezes the snapshot; statement_timestamp() is when it did.
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0), statement_timestamp() FROM ledger_journals WHERE environment = $1`, env).Scan(&snap.Head, &snap.TakenAt); err != nil {
		return LiabilitySnapshot{}, err
	}
	// Same query as the audit recipe (docs/cre/OPERATIONS.md): an inner join, so an account opened after the
	// snapshot with no counted lines never adds an asset.
	rows, err := tx.QueryContext(ctx, `SELECT a.asset, COALESCE(SUM(l.amount), 0)::text
FROM ledger_accounts a
JOIN ledger_lines l ON l.account_id = a.id
WHERE a.environment = $1 AND a.owner_type = $2 AND a.kind = $3 AND l.journal_id <= $4
GROUP BY a.asset
ORDER BY a.asset COLLATE "C"`, env, string(OwnerMember), string(KindLiability), snap.Head)
	if err != nil {
		return LiabilitySnapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var asset, signed string
		if err := rows.Scan(&asset, &signed); err != nil {
			return LiabilitySnapshot{}, err
		}
		total, err := decimal.NewFromString(signed)
		if err != nil {
			return LiabilitySnapshot{}, err
		}
		snap.Totals = append(snap.Totals, LiabilityTotal{Asset: asset, Total: NaturalBalance(KindLiability, total)})
	}
	if err := rows.Err(); err != nil {
		return LiabilitySnapshot{}, err
	}
	snap.TakenAt = snap.TakenAt.UTC()
	return snap, tx.Commit()
}
