package databaseexplorer

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExplorer_QuerySecurityContract(t *testing.T) {
	t.Parallel()

	adapter := NewMemoryAdapter([]MemoryRow{{
		Dataset:  DatasetPaymentLifecycle,
		TenantID: "tenant-b",
		Values: Row{
			FieldPaymentReference: {Value: "pay_b_1", Classification: ClassificationInternal},
			FieldCustomerEmail:    {Value: "customer@example.test", Classification: ClassificationPII},
			FieldWebhookSecret:    {Value: "never-return", Classification: ClassificationSecret},
		},
	}})
	explorer := New(adapter, Config{MaxRows: 100, QueryTimeout: time.Second})

	t.Run("cross tenant merchant is denied", func(t *testing.T) {
		_, err := explorer.Query(context.Background(), Query{
			Dataset:  DatasetPaymentLifecycle,
			Actor:    Actor{Role: RoleMerchant, TenantID: "tenant-a"},
			TenantID: "tenant-b",
			Limit:    25,
		})
		if !errors.Is(err, ErrTenantDenied) {
			t.Fatalf("Query() error = %v, want ErrTenantDenied", err)
		}
	})

	t.Run("unknown dataset filter and sort fail closed", func(t *testing.T) {
		base := Query{Dataset: DatasetPaymentLifecycle, Actor: Actor{Role: RoleMerchant, TenantID: "tenant-b"}, TenantID: "tenant-b", Limit: 25}
		tests := []struct {
			name  string
			query Query
			want  error
		}{
			{name: "dataset", query: Query{Dataset: Dataset("raw_sql"), Actor: base.Actor, TenantID: base.TenantID}, want: ErrUnknownDataset},
			{name: "filter", query: withFilters(base, Filter{Field: FieldWebhookSecret, Operator: OperatorEqual, Value: "x"}), want: ErrUnknownFilter},
			{name: "sort", query: withSort(base, Sort{Field: FieldCustomerEmail, Direction: SortAscending}), want: ErrUnknownSort},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := explorer.Query(context.Background(), tt.query)
				if !errors.Is(err, tt.want) {
					t.Fatalf("Query() error = %v, want %v", err, tt.want)
				}
			})
		}
	})

	t.Run("maximum page size is enforced", func(t *testing.T) {
		_, err := explorer.Query(context.Background(), Query{
			Dataset: DatasetPaymentLifecycle, Actor: Actor{Role: RoleMerchant, TenantID: "tenant-b"}, TenantID: "tenant-b", Limit: 101,
		})
		if !errors.Is(err, ErrLimitExceeded) {
			t.Fatalf("Query() error = %v, want ErrLimitExceeded", err)
		}
	})

	t.Run("caller cancellation wins", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := explorer.Query(ctx, Query{
			Dataset: DatasetPaymentLifecycle, Actor: Actor{Role: RoleMerchant, TenantID: "tenant-b"}, TenantID: "tenant-b", Limit: 25,
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Query() error = %v, want context.Canceled", err)
		}
	})

	t.Run("PII is redacted and secret fields are absent", func(t *testing.T) {
		page, err := explorer.Query(context.Background(), Query{
			Dataset: DatasetPaymentLifecycle, Actor: Actor{Role: RoleMerchant, TenantID: "tenant-b"}, TenantID: "tenant-b", Limit: 25,
		})
		if err != nil {
			t.Fatalf("Query() error = %v", err)
		}
		if got := page.Rows[0][FieldCustomerEmail]; got.Value != "c***@example.test" || !got.Redacted {
			t.Fatalf("customer email = %#v, want masked PII", got)
		}
		if _, exists := page.Rows[0][FieldWebhookSecret]; exists {
			t.Fatal("secret field was returned")
		}
	})
}

func withFilters(query Query, filters ...Filter) Query {
	query.Filters = filters
	return query
}

func withSort(query Query, sort Sort) Query {
	query.Sort = sort
	return query
}
