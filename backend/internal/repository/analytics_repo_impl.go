package repository

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// bucketExpr returns the dialect-specific SQL expression for grouping
// timestamps into day/week/month buckets. SQLite uses strftime; Postgres
// uses to_char; everything else falls back to DATE() which is a per-day
// truncation.
//
// The output is interpolated directly into the SQL string — never user
// input — so there is no injection risk.
func (r *AnalyticsRepositoryImpl) bucketExpr(interval string) string {
	dialect := r.db.Dialector.Name()

	switch dialect {
	case "sqlite":
		switch interval {
		case "week":
			return "strftime('%Y-%W', d.created_at)"
		case "month":
			return "strftime('%Y-%m', d.created_at)"
		default:
			return "strftime('%Y-%m-%d', d.created_at)"
		}
	case "postgres":
		switch interval {
		case "week":
			return "to_char(d.created_at, 'IYYY-IW')"
		case "month":
			return "to_char(d.created_at, 'YYYY-MM')"
		default:
			return "to_char(d.created_at, 'YYYY-MM-DD')"
		}
	default:
		// MySQL / MariaDB / unknown — DATE() drops the time part.
		return "DATE(d.created_at)"
	}
}

// silence "fmt unused" warning if a future change removes the only Sprintf call
var _ = fmt.Sprintf

// VolumeBucket is a single time-bucket returned by GetVolumeOverTime.
type VolumeBucket struct {
	// Bucket is the start of the time bucket (day/hour/week depending on interval).
	Bucket time.Time
	// BucketLabel is the string representation for display.
	BucketLabel string
	// Volume is the total confirmed deposit amount in that bucket.
	Volume decimal.Decimal
	// PaymentCount is the number of distinct payments in that bucket.
	PaymentCount int
}

// CustomerSummary is a row returned by GetTopCustomers.
type CustomerSummary struct {
	// CustomerID is the merchant-supplied customer identifier.
	CustomerID string
	// CustomerEmail is the optional email address on record.
	CustomerEmail *string
	// TotalVolume is the sum of all confirmed deposits from this customer.
	TotalVolume decimal.Decimal
	// PaymentCount is the number of payments from this customer.
	PaymentCount int
}

// RevenueBreakdown is a row returned by GetRevenueBreakdown.
type RevenueBreakdown struct {
	// BlockchainCode is the chain code (ETH, BTC, …).
	BlockchainCode string
	// CurrencyCode is the token code (USDT, ETH, …).
	CurrencyCode string
	// State is the payment state (FILLED, OPEN, …).
	State string
	// TotalAmount is the sum of amounts in this bucket.
	TotalAmount decimal.Decimal
	// Count is the number of payments in this bucket.
	Count int
}

// SweepStats is returned by GetSweepStats.
type SweepStats struct {
	// TotalSweeps is the count of completed sweeps.
	TotalSweeps int
	// TotalSwept is the sum of amounts swept.
	TotalSwept decimal.Decimal
	// TotalGas is the sum of gas fees spent.
	TotalGas decimal.Decimal
}

// WithdrawalStats is returned by GetWithdrawalStats.
type WithdrawalStats struct {
	// TotalWithdrawals is the count of completed withdrawals.
	TotalWithdrawals int
	// TotalAmount is the sum of withdrawal amounts.
	TotalAmount decimal.Decimal
	// TotalGas is the sum of gas fees paid.
	TotalGas decimal.Decimal
}

// DashboardSummary is the all-up aggregate for the dashboard home tile.
type DashboardSummary struct {
	// TotalPayments is the lifetime count of payment requests.
	TotalPayments int64
	// FilledPayments is the count of payments in FILLED state.
	FilledPayments int64
	// TotalVolume is the sum of deposit amounts across all FILLED payments.
	TotalVolume decimal.Decimal
	// TotalWithdrawals is the lifetime count of withdrawals.
	TotalWithdrawals int64
	// ActiveWebhooks is the count of active webhook endpoints.
	ActiveWebhooks int64
}

// AnalyticsRepository defines read-only aggregation queries used by
// AnalyticsService. All methods are scoped to a single platform via platformID
// for tenant isolation.
type AnalyticsRepository interface {
	// GetVolumeOverTime returns time-bucketed confirmed deposit volume for the
	// given platform. The interval parameter is "day", "week", or "month".
	GetVolumeOverTime(platformID uint, start, end time.Time, interval string) ([]VolumeBucket, error)

	// GetTopCustomers returns the top N customers by deposit volume.
	GetTopCustomers(platformID uint, limit int) ([]CustomerSummary, error)

	// GetRevenueBreakdown returns payment totals grouped by chain, currency, and state.
	GetRevenueBreakdown(platformID uint, start, end time.Time) ([]RevenueBreakdown, error)

	// GetSweepStats returns aggregate sweep statistics.
	GetSweepStats(platformID uint, start, end time.Time) (SweepStats, error)

	// GetWithdrawalStats returns aggregate withdrawal statistics.
	GetWithdrawalStats(platformID uint, start, end time.Time) (WithdrawalStats, error)

	// GetDashboardSummary returns the all-up counts and totals for the dashboard.
	GetDashboardSummary(platformID uint) (DashboardSummary, error)
}

// AnalyticsRepositoryImpl is the GORM-backed implementation of AnalyticsRepository.
type AnalyticsRepositoryImpl struct {
	db *gorm.DB
}

// NewAnalyticsRepository constructs a new AnalyticsRepository.
func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &AnalyticsRepositoryImpl{db: db}
}

// GetVolumeOverTime returns time-bucketed confirmed deposit volume. SQLite uses
// strftime for grouping; Postgres uses to_char. We detect the dialect via the
// driver name and branch accordingly.
func (r *AnalyticsRepositoryImpl) GetVolumeOverTime(platformID uint, start, end time.Time, interval string) ([]VolumeBucket, error) {
	type row struct {
		BucketLabel  string
		Volume       decimal.Decimal
		PaymentCount int
	}

	// Bucket expression varies by SQL dialect:
	//   - SQLite (tests): strftime('format', col)
	//   - Postgres (prod): to_char(col, 'format')
	// All other inputs default to DATE(col) which works in MySQL too.
	bucketExpr := r.bucketExpr(interval)

	var rows []row
	query := fmt.Sprintf(`
		SELECT
			%s AS bucket_label,
			COALESCE(SUM(d.amount), 0)   AS volume,
			COUNT(DISTINCT d.payment_request_id) AS payment_count
		FROM deposits d
		JOIN payment_requests p ON p.id = d.payment_request_id
		WHERE p.external_platform_id = ?
		  AND d.status = 'confirmed'
		  AND d.created_at >= ?
		  AND d.created_at < ?
		  AND d.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		GROUP BY bucket_label
		ORDER BY bucket_label ASC
	`, bucketExpr)
	err := r.db.Raw(query, platformID, start, end).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	buckets := make([]VolumeBucket, len(rows))
	for i, r := range rows {
		buckets[i] = VolumeBucket{
			BucketLabel:  r.BucketLabel,
			Volume:       r.Volume,
			PaymentCount: r.PaymentCount,
		}
	}
	return buckets, nil
}

// GetTopCustomers returns the top N customers by total deposit volume for a platform.
func (r *AnalyticsRepositoryImpl) GetTopCustomers(platformID uint, limit int) ([]CustomerSummary, error) {
	if limit <= 0 {
		limit = 10
	}

	type row struct {
		CustomerID    string
		CustomerEmail *string
		TotalVolume   decimal.Decimal
		PaymentCount  int
	}

	var rows []row
	err := r.db.Raw(`
		SELECT
			p.customer_id          AS customer_id,
			p.customer_email       AS customer_email,
			COALESCE(SUM(d.amount), 0) AS total_volume,
			COUNT(DISTINCT p.id)   AS payment_count
		FROM payment_requests p
		JOIN deposits d ON d.payment_request_id = p.id AND d.deleted_at IS NULL
		WHERE p.external_platform_id = ?
		  AND p.customer_id IS NOT NULL
		  AND d.status = 'confirmed'
		  AND p.deleted_at IS NULL
		GROUP BY p.customer_id, p.customer_email
		ORDER BY total_volume DESC
		LIMIT ?
	`, platformID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	summaries := make([]CustomerSummary, len(rows))
	for i, r := range rows {
		summaries[i] = CustomerSummary{
			CustomerID:    r.CustomerID,
			CustomerEmail: r.CustomerEmail,
			TotalVolume:   r.TotalVolume,
			PaymentCount:  r.PaymentCount,
		}
	}
	return summaries, nil
}

// GetRevenueBreakdown returns payment totals grouped by blockchain, currency, and state.
func (r *AnalyticsRepositoryImpl) GetRevenueBreakdown(platformID uint, start, end time.Time) ([]RevenueBreakdown, error) {
	type row struct {
		BlockchainCode string
		CurrencyCode   string
		State          string
		TotalAmount    decimal.Decimal
		Count          int
	}

	var rows []row
	err := r.db.Raw(`
		SELECT
			COALESCE(b.code, 'unknown')  AS blockchain_code,
			COALESCE(c.code, 'unknown')  AS currency_code,
			p.state,
			COALESCE(SUM(d.amount), 0)   AS total_amount,
			COUNT(DISTINCT p.id)          AS count
		FROM payment_requests p
		LEFT JOIN deposits d ON d.payment_request_id = p.id AND d.deleted_at IS NULL AND d.status IN ('confirmed', 'swept')
		LEFT JOIN deposit_addresses da ON da.id = p.deposit_address_id AND da.deleted_at IS NULL
		LEFT JOIN blockchain_currencies bc ON bc.id = da.blockchain_currency_id AND bc.deleted_at IS NULL
		LEFT JOIN blockchains b ON b.id = bc.blockchain_id AND b.deleted_at IS NULL
		LEFT JOIN currencies c ON c.id = bc.currency_id AND c.deleted_at IS NULL
		WHERE p.external_platform_id = ?
		  AND p.created_at >= ?
		  AND p.created_at < ?
		  AND p.deleted_at IS NULL
		GROUP BY b.code, c.code, p.state
		ORDER BY total_amount DESC
	`, platformID, start, end).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]RevenueBreakdown, len(rows))
	for i, r := range rows {
		result[i] = RevenueBreakdown{
			BlockchainCode: r.BlockchainCode,
			CurrencyCode:   r.CurrencyCode,
			State:          r.State,
			TotalAmount:    r.TotalAmount,
			Count:          r.Count,
		}
	}
	return result, nil
}

// GetSweepStats returns aggregate sweep statistics across the entire instance.
//
// INTERNAL USE ONLY — sweeps are not directly scoped to a platform in the
// current data model, so this method returns instance-wide stats. It must
// not be exposed to per-merchant API endpoints; doing so leaks cross-tenant
// volume + gas data. Phase J adds a join through deposit_addresses →
// payment_requests.external_platform_id and re-introduces a scoped variant.
func (r *AnalyticsRepositoryImpl) GetSweepStats(_ uint, start, end time.Time) (SweepStats, error) {
	type row struct {
		TotalSweeps int
		TotalSwept  decimal.Decimal
		TotalGas    decimal.Decimal
	}
	var result row
	err := r.db.Raw(`
		SELECT
			COUNT(*) AS total_sweeps,
			COALESCE(SUM(total_amount), 0) AS total_swept,
			COALESCE(SUM(total_gas_fee), 0) AS total_gas
		FROM sweeps
		WHERE status = 'completed'
		  AND created_at >= ?
		  AND created_at < ?
		  AND deleted_at IS NULL
	`, start, end).Scan(&result).Error
	if err != nil {
		return SweepStats{}, err
	}
	return SweepStats{
		TotalSweeps: result.TotalSweeps,
		TotalSwept:  result.TotalSwept,
		TotalGas:    result.TotalGas,
	}, nil
}

// GetWithdrawalStats returns aggregate withdrawal statistics.
func (r *AnalyticsRepositoryImpl) GetWithdrawalStats(platformID uint, start, end time.Time) (WithdrawalStats, error) {
	type row struct {
		TotalWithdrawals int
		TotalAmount      decimal.Decimal
		TotalGas         decimal.Decimal
	}
	var result row
	err := r.db.Raw(`
		SELECT
			COUNT(*) AS total_withdrawals,
			COALESCE(SUM(amount), 0)        AS total_amount,
			COALESCE(SUM(network_fee), 0)   AS total_gas
		FROM withdrawals
		WHERE external_platform_id = ?
		  AND state = 'processed'
		  AND created_at >= ?
		  AND created_at < ?
		  AND deleted_at IS NULL
	`, platformID, start, end).Scan(&result).Error
	if err != nil {
		return WithdrawalStats{}, err
	}
	return WithdrawalStats{
		TotalWithdrawals: result.TotalWithdrawals,
		TotalAmount:      result.TotalAmount,
		TotalGas:         result.TotalGas,
	}, nil
}

// GetDashboardSummary returns the all-up counts and totals for the dashboard
// home tile. Each query checks its own error so a single failed scan no
// longer silently zeros the response.
//
// Sweeps batch addresses across platforms, so no sweep count is reported per platform.
func (r *AnalyticsRepositoryImpl) GetDashboardSummary(platformID uint) (DashboardSummary, error) {
	var summary DashboardSummary

	// Payment counts.
	if err := r.db.Raw(
		`SELECT COUNT(*) FROM payment_requests WHERE external_platform_id = ? AND deleted_at IS NULL`,
		platformID,
	).Scan(&summary.TotalPayments).Error; err != nil {
		return summary, fmt.Errorf("dashboard total payments: %w", err)
	}
	if err := r.db.Raw(
		`SELECT COUNT(*) FROM payment_requests WHERE external_platform_id = ? AND state = 'FILLED' AND deleted_at IS NULL`,
		platformID,
	).Scan(&summary.FilledPayments).Error; err != nil {
		return summary, fmt.Errorf("dashboard filled payments: %w", err)
	}

	// Volume from confirmed deposits.
	type volRow struct{ Volume decimal.Decimal }
	var vr volRow
	if err := r.db.Raw(`
		SELECT COALESCE(SUM(d.amount), 0) AS volume
		FROM deposits d
		JOIN payment_requests p ON p.id = d.payment_request_id
		WHERE p.external_platform_id = ?
		  AND d.status = 'confirmed'
		  AND d.deleted_at IS NULL
		  AND p.deleted_at IS NULL
	`, platformID).Scan(&vr).Error; err != nil {
		return summary, fmt.Errorf("dashboard total volume: %w", err)
	}
	summary.TotalVolume = vr.Volume

	// Withdrawal count.
	if err := r.db.Raw(
		`SELECT COUNT(*) FROM withdrawals WHERE external_platform_id = ? AND deleted_at IS NULL`,
		platformID,
	).Scan(&summary.TotalWithdrawals).Error; err != nil {
		return summary, fmt.Errorf("dashboard total withdrawals: %w", err)
	}

	// Active webhooks.
	if err := r.db.Raw(
		`SELECT COUNT(*) FROM webhooks WHERE external_platform_id = ? AND active = true AND deleted_at IS NULL`,
		platformID,
	).Scan(&summary.ActiveWebhooks).Error; err != nil {
		return summary, fmt.Errorf("dashboard active webhooks: %w", err)
	}

	return summary, nil
}
