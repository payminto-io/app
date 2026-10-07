// Package storetest is the contract every links.Store must pass; links runs it on MemStore, and on
// PGStore behind the integration tag.
package storetest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/shopspring/decimal"
)

// Fixture is a fresh store plus a member and platform it may reference; Webhook belongs to Platform.
type Fixture struct {
	Store    links.Store
	Member   uint
	Platform uint
	Webhook  uint
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func link(f Fixture) links.Link {
	in := links.DefaultInput()
	in.Title, in.Currency = "Beans", "USD"
	amount := dec("25")
	in.Amount = &amount
	in.Methods = []links.MethodSpec{{Method: fees.MethodCard}}
	in.Metadata = map[string]string{"k": "v"}
	in.LineItems = []links.LineItem{}
	in.Questions = []links.Question{{Key: "size", Label: "Size", Type: links.QuestionSelect, Options: []string{"S", "M"}, Required: true, PerOrder: true}}
	return links.Link{Input: in, Total: &amount, MemberID: f.Member, ExternalPlatformID: f.Platform, Environment: links.EnvTest, Status: links.StatusDraft}
}

func use(l links.Link, key string, now time.Time) links.LinkPayment {
	return links.LinkPayment{
		LinkID: l.ID, IdempotencyKey: key, RequestHash: fmt.Sprintf("%064d", 0), Environment: l.Environment,
		Method: links.MethodSpec{Method: fees.MethodCard}, Amount: dec("25"), Currency: "USD", CustomerTotal: dec("25"),
		FeeBearer: l.FeeBearer, ClientKey: "client-a", ReservedUntil: now.Add(time.Minute), OpenUntil: now.Add(time.Minute),
		Answers: []links.Answer{{QuestionKey: "size", QuestionLabel: "Size", Value: "M"}},
	}
}

func active(t *testing.T, f Fixture, edit func(*links.Link)) links.Link {
	t.Helper()
	ctx := context.Background()
	l := link(f)
	if edit != nil {
		edit(&l)
	}
	l, err := f.Store.Insert(ctx, l)
	if err != nil {
		t.Fatal(err)
	}
	code := fmt.Sprintf("C%011d", time.Now().UnixNano()%1e11)
	l, err = f.Store.SetStatus(ctx, f.Platform, l.ID, l.Revision, links.StatusActive, code, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// Run executes the contract; newFixture must return an isolated store each call.
func Run(t *testing.T, newFixture func(t *testing.T) Fixture) {
	ctx := context.Background()
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

	t.Run("insert get scope and revision", func(t *testing.T) {
		f := newFixture(t)
		l, err := f.Store.Insert(ctx, link(f))
		if err != nil || l.ID == "" || l.Revision != 1 || len(l.Questions) != 1 || l.Metadata["k"] != "v" {
			t.Fatalf("insert %+v %v", l, err)
		}
		if _, err := f.Store.Get(ctx, f.Platform+1000, l.ID); !errors.Is(err, links.ErrStoreNotFound) {
			t.Fatalf("other platform: %v", err)
		}
		saved, err := f.Store.Save(ctx, l)
		if err != nil || saved.Revision != 2 {
			t.Fatalf("save %+v %v", saved.Revision, err)
		}
		if _, err := f.Store.Save(ctx, l); !errors.Is(err, links.ErrStale) {
			t.Fatalf("save on an old revision: %v", err)
		}
		if _, err := f.Store.SetStatus(ctx, f.Platform, l.ID, l.Revision, links.StatusArchived, "", now); !errors.Is(err, links.ErrStale) {
			t.Fatalf("status on an old revision: %v", err)
		}
		if err := f.Store.DeleteDraft(ctx, f.Platform+1000, l.ID, saved.Revision); !errors.Is(err, links.ErrStoreNotFound) {
			t.Fatalf("delete from another platform: %v", err)
		}
	})

	t.Run("short codes are unique and minted once", func(t *testing.T) {
		f := newFixture(t)
		a, _ := f.Store.Insert(ctx, link(f))
		b, _ := f.Store.Insert(ctx, link(f))
		a, err := f.Store.SetStatus(ctx, f.Platform, a.ID, a.Revision, links.StatusActive, "SAMECODE0001", now)
		if err != nil || a.ShortCode != "SAMECODE0001" || a.PublishedAt == nil {
			t.Fatalf("publish %+v %v", a, err)
		}
		if _, err := f.Store.SetStatus(ctx, f.Platform, b.ID, b.Revision, links.StatusActive, "SAMECODE0001", now); !errors.Is(err, links.ErrShortCodeTaken) {
			t.Fatalf("taken code: %v", err)
		}
		paused, _ := f.Store.SetStatus(ctx, f.Platform, a.ID, a.Revision, links.StatusPaused, "", now)
		resumed, err := f.Store.SetStatus(ctx, f.Platform, a.ID, paused.Revision, links.StatusActive, "OTHERCODE002", now.Add(time.Hour))
		if err != nil || resumed.ShortCode != "SAMECODE0001" || !resumed.PublishedAt.Equal(*a.PublishedAt) {
			t.Fatalf("resume %+v %v", resumed, err)
		}
		got, err := f.Store.GetByShortCode(ctx, "SAMECODE0001")
		if err != nil || got.ID != a.ID {
			t.Fatalf("by code %v", err)
		}
	})

	t.Run("reserve complete release and replay", func(t *testing.T) {
		f := newFixture(t)
		l := active(t, f, func(l *links.Link) { l.MultiUse = true })
		r, existing, err := f.Store.Reserve(ctx, use(l, "k1", now), now, links.ReserveLimits{})
		if err != nil || existing != nil || r.ID == "" {
			t.Fatalf("reserve %+v %v %v", r, existing, err)
		}
		_, existing, err = f.Store.Reserve(ctx, use(l, "k1", now), now, links.ReserveLimits{})
		if err != nil || existing == nil || existing.ID != r.ID || len(existing.Answers) != 1 {
			t.Fatalf("same key %+v %v", existing, err)
		}
		if err := f.Store.Complete(ctx, r.ID, links.CreatedPayment{Reference: "ref-1"}, now.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := f.Store.Complete(ctx, r.ID, links.CreatedPayment{Reference: "ref-2"}, now); !errors.Is(err, links.ErrStale) {
			t.Fatalf("second complete: %v", err)
		}
		if err := f.Store.Release(ctx, r.ID); !errors.Is(err, links.ErrStale) {
			t.Fatalf("release of a created use: %v", err)
		}
		got, err := f.Store.FindPayment(ctx, l.ID, "k1")
		if err != nil || got.PaymentReference != "ref-1" || got.Processor == nil || got.Processor.Reference != "ref-1" {
			t.Fatalf("find %+v %v", got, err)
		}
		r2, _, _ := f.Store.Reserve(ctx, use(l, "k2", now), now, links.ReserveLimits{})
		if err := f.Store.Release(ctx, r2.ID); err != nil {
			t.Fatal(err)
		}
		cur, _ := f.Store.Get(ctx, f.Platform, l.ID)
		if cur.UsesCount != 1 {
			t.Fatalf("uses %d after one completed and one released", cur.UsesCount)
		}
		if got, _ := f.Store.FindPayment(ctx, l.ID, "k2"); got != nil {
			t.Fatal("released use is still findable")
		}
		again, existing, err := f.Store.Reserve(ctx, use(l, "k2", now), now, links.ReserveLimits{})
		if err != nil || existing != nil || again.ID == r2.ID {
			t.Fatalf("a released key cannot be reserved afresh: %+v %v", existing, err)
		}
		if err := f.Store.Release(ctx, r2.ID); !errors.Is(err, links.ErrStale) {
			t.Fatalf("second release: %v", err)
		}
		if got, err := f.Store.Link(ctx, l.ID); err != nil || got.ID != l.ID {
			t.Fatalf("unscoped link read %v", err)
		}
	})

	t.Run("reserve enforces availability limits and fee bearer", func(t *testing.T) {
		f := newFixture(t)
		single := active(t, f, nil)
		if _, _, err := f.Store.Reserve(ctx, use(single, "a", now), now, links.ReserveLimits{}); err != nil {
			t.Fatal(err)
		}
		if _, _, err := f.Store.Reserve(ctx, use(single, "b", now), now, links.ReserveLimits{}); links.CodeOf(err) != links.CodeUseLimitReached {
			t.Fatalf("second use of a single-use link: %v", err)
		}
		exp := now.Add(time.Minute)
		expiring := active(t, f, func(l *links.Link) { l.ExpiresAt = &exp })
		if _, _, err := f.Store.Reserve(ctx, use(expiring, "a", exp), exp, links.ReserveLimits{}); links.CodeOf(err) != links.CodeExpired {
			t.Fatalf("expired: %v", err)
		}
		multi := active(t, f, func(l *links.Link) { l.MultiUse = true })
		wrongBearer := use(multi, "x", now)
		wrongBearer.FeeBearer = fees.BearerCustomer
		if _, _, err := f.Store.Reserve(ctx, wrongBearer, now, links.ReserveLimits{}); !errors.Is(err, links.ErrStale) {
			t.Fatalf("fee bearer changed under the quote: %v", err)
		}
		caps := links.ReserveLimits{MaxOpen: 3, MaxOpenPerClient: 2}
		for i, want := range []links.Code{"", "", links.CodeOpenPaymentsLimit} {
			if _, _, err := f.Store.Reserve(ctx, use(multi, fmt.Sprint("c", i), now), now, caps); links.CodeOf(err) != want {
				t.Fatalf("client cap %d: %v", i, err)
			}
		}
		other := use(multi, "o1", now)
		other.ClientKey = "client-b"
		if _, _, err := f.Store.Reserve(ctx, other, now, caps); err != nil {
			t.Fatal(err)
		}
		other.IdempotencyKey, other.ClientKey = "o2", "client-c"
		if _, _, err := f.Store.Reserve(ctx, other, now, caps); links.CodeOf(err) != links.CodeOpenPaymentsLimit {
			t.Fatalf("link cap: %v", err)
		}
		later := now.Add(2 * time.Minute)
		other.IdempotencyKey = "o3"
		if _, _, err := f.Store.Reserve(ctx, other, later, caps); err != nil {
			t.Fatalf("caps after the uses stopped being open: %v", err)
		}
	})

	t.Run("finished uses stop counting as open", func(t *testing.T) {
		f := newFixture(t)
		l := active(t, f, func(l *links.Link) { l.MultiUse = true })
		var ids []string
		for i := range 2 {
			r, _, err := f.Store.Reserve(ctx, use(l, fmt.Sprint("o", i), now), now, links.ReserveLimits{})
			if err != nil {
				t.Fatal(err)
			}
			if err := f.Store.Complete(ctx, r.ID, links.CreatedPayment{Reference: r.ID}, now.Add(time.Hour)); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, r.ID)
		}
		open, err := f.Store.OpenCreated(ctx, l.ID, now, 10)
		if err != nil || len(open) != 2 {
			t.Fatalf("open %d %v", len(open), err)
		}
		if err := f.Store.CloseUses(ctx, ids[:1], now); err != nil {
			t.Fatal(err)
		}
		open, _ = f.Store.OpenCreated(ctx, l.ID, now, 10)
		if len(open) != 1 || open[0].ID != ids[1] {
			t.Fatalf("after close %+v", open)
		}
		caps := links.ReserveLimits{MaxOpenPerClient: 2}
		if _, _, err := f.Store.Reserve(ctx, use(l, "o2", now), now, caps); err != nil {
			t.Fatalf("a closed use still counted: %v", err)
		}
	})

	t.Run("expired pending uses are listed oldest first", func(t *testing.T) {
		f := newFixture(t)
		l := active(t, f, func(l *links.Link) { l.MultiUse = true })
		for i, lease := range []time.Duration{3 * time.Minute, time.Minute, 2 * time.Minute, time.Hour} {
			u := use(l, fmt.Sprint("p", i), now)
			u.ReservedUntil = now.Add(lease)
			if _, _, err := f.Store.Reserve(ctx, u, now, links.ReserveLimits{}); err != nil {
				t.Fatal(err)
			}
		}
		got, err := f.Store.ExpiredPending(ctx, now.Add(10*time.Minute), 2)
		if err != nil || len(got) != 2 || got[0].IdempotencyKey != "p1" || got[1].IdempotencyKey != "p2" {
			t.Fatalf("expired %+v %v", got, err)
		}
	})

	t.Run("one winner on the last use", func(t *testing.T) {
		f := newFixture(t)
		limit := 2
		l := active(t, f, func(l *links.Link) { l.MultiUse, l.UseLimit = true, &limit })
		var wg sync.WaitGroup
		errs := make([]error, 16)
		for i := range errs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, _, errs[i] = f.Store.Reserve(ctx, use(l, fmt.Sprint("r", i), now), now, links.ReserveLimits{})
			}(i)
		}
		wg.Wait()
		wins := 0
		for _, err := range errs {
			switch {
			case err == nil:
				wins++
			case links.CodeOf(err) != links.CodeUseLimitReached:
				t.Fatalf("racer: %v", err)
			}
		}
		if wins != 2 {
			t.Fatalf("%d winners for a limit of 2", wins)
		}
	})
}
