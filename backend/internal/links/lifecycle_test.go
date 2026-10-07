package links

import (
	"context"
	"fmt"
	"testing"

	"github.com/payminto/payminto/backend/internal/fees"
)

// linkIn puts a fresh link in the given status.
func (f *fixture) linkIn(st Status) Link {
	f.t.Helper()
	ctx, pid := context.Background(), merchant.PlatformID
	l := f.create(validInput())
	var err error
	switch st {
	case StatusActive:
		l, err = f.svc.Publish(ctx, pid, l.ID)
	case StatusPaused:
		if l, err = f.svc.Publish(ctx, pid, l.ID); err == nil {
			l, err = f.svc.Pause(ctx, pid, l.ID)
		}
	case StatusArchived:
		l, err = f.svc.Archive(ctx, pid, l.ID)
	}
	if err != nil {
		f.t.Fatalf("put link in %s: %v", st, err)
	}
	return l
}

func TestLifecycleTransitions(t *testing.T) {
	type op struct {
		name string
		do   func(*Service, string) (Link, error)
	}
	pid := merchant.PlatformID
	ops := []op{
		{"publish", func(s *Service, id string) (Link, error) { return s.Publish(context.Background(), pid, id) }},
		{"pause", func(s *Service, id string) (Link, error) { return s.Pause(context.Background(), pid, id) }},
		{"archive", func(s *Service, id string) (Link, error) { return s.Archive(context.Background(), pid, id) }},
	}
	want := map[Status]map[string]Status{
		StatusDraft:    {"publish": StatusActive, "archive": StatusArchived},
		StatusActive:   {"pause": StatusPaused, "archive": StatusArchived},
		StatusPaused:   {"publish": StatusActive, "archive": StatusArchived},
		StatusArchived: {},
	}
	for from, allowed := range want {
		for _, o := range ops {
			t.Run(fmt.Sprintf("%s %s", o.name, from), func(t *testing.T) {
				f := newFixture(t)
				l := f.linkIn(from)
				got, err := o.do(f.svc, l.ID)
				to, ok := allowed[o.name]
				if !ok {
					if CodeOf(err) != CodeInvalidTransition {
						t.Fatalf("err %v, want invalid_transition", err)
					}
					return
				}
				if err != nil || got.Status != to {
					t.Fatalf("got %s, %v; want %s", got.Status, err, to)
				}
			})
		}
	}
}

func TestShortCodeMintedOnPublishAndKeptOnResume(t *testing.T) {
	f := newFixture(t)
	ctx, pid := context.Background(), merchant.PlatformID
	l := f.published(validInput())
	if !ValidShortCode(l.ShortCode) || len(l.ShortCode) != ShortCodeLength {
		t.Fatalf("short code %q", l.ShortCode)
	}
	if l.PublishedAt == nil || !l.PublishedAt.Equal(f.now) {
		t.Fatalf("published_at %v", l.PublishedAt)
	}
	paused, _ := f.svc.Pause(ctx, pid, l.ID)
	resumed, err := f.svc.Publish(ctx, pid, paused.ID)
	if err != nil || resumed.ShortCode != l.ShortCode {
		t.Fatalf("resume changed the short code: %q -> %q (%v)", l.ShortCode, resumed.ShortCode, err)
	}
	if got := f.svc.URL(l.ShortCode); got != "https://checkout.test/l/"+l.ShortCode {
		t.Fatalf("url %s", got)
	}
}

func TestPublishRetriesAShortCodeCollision(t *testing.T) {
	codes := []string{"AAAAAAAAAAAA", "AAAAAAAAAAAA", "BBBBBBBBBBBB"}
	i := 0
	f := newFixture(t, WithShortCodes(func() (string, error) { c := codes[i]; i++; return c, nil }))
	first := f.published(validInput())
	second := f.published(validInput())
	if first.ShortCode != "AAAAAAAAAAAA" || second.ShortCode != "BBBBBBBBBBBB" {
		t.Fatalf("codes %s %s", first.ShortCode, second.ShortCode)
	}
}

func TestPublishGivesUpAfterRepeatedCollisions(t *testing.T) {
	f := newFixture(t, WithShortCodes(func() (string, error) { return "AAAAAAAAAAAA", nil }))
	f.published(validInput())
	l := f.create(validInput())
	_, err := f.svc.Publish(context.Background(), merchant.PlatformID, l.ID)
	if CodeOf(err) != CodeShortCodeExhausted {
		t.Fatalf("err %v", err)
	}
}

func TestPublishedLinkKeepsMoneyFields(t *testing.T) {
	cases := []struct {
		field string
		edit  func(*Input)
	}{
		{"amount", func(in *Input) { in.Amount = decp("30.00") }},
		{"currency", func(in *Input) { in.Currency = "EUR" }},
		{"methods", func(in *Input) { in.Methods = []MethodSpec{card, upi} }},
		{"amount_mode", func(in *Input) {
			in.AmountMode, in.Amount = AmountLineItems, nil
			in.LineItems = []LineItem{{Name: "a", Quantity: 1, UnitPrice: dec("25")}}
		}},
	}
	for _, st := range []Status{StatusActive, StatusPaused} {
		for _, tc := range cases {
			t.Run(string(st)+" "+tc.field, func(t *testing.T) {
				f := newFixture(t)
				l := f.linkIn(st)
				in := l.Input
				tc.edit(&in)
				_, err := f.svc.Update(context.Background(), merchant.PlatformID, l.ID, in)
				if e, ok := err.(*Error); !ok || e.Code != CodePublishedImmutable || e.Field != tc.field {
					t.Fatalf("err %v, want link_published_immutable on %s", err, tc.field)
				}
			})
		}
	}
}

func TestPublishedLineItemsAreFixed(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.AmountMode, in.Amount = AmountLineItems, nil
	in.LineItems = []LineItem{{Name: "Beans", Quantity: 2, UnitPrice: dec("10")}}
	l := f.published(in)
	next := l.Input
	next.LineItems = []LineItem{{Name: "Beans", Quantity: 3, UnitPrice: dec("10")}}
	_, err := f.svc.Update(context.Background(), merchant.PlatformID, l.ID, next)
	if e, ok := err.(*Error); !ok || e.Field != "line_items" {
		t.Fatalf("err %v", err)
	}
}

func TestPublishedLinkEditsOtherFieldsAndRevalidates(t *testing.T) {
	f := newFixture(t)
	ctx, pid := context.Background(), merchant.PlatformID
	in := validInput()
	in.Methods = []MethodSpec{upi}
	l := f.published(in)
	next := l.Input
	next.Title = "Coffee beans, roasted"
	updated, err := f.svc.Update(ctx, pid, l.ID, next)
	if err != nil || updated.Title != "Coffee beans, roasted" || updated.Status != StatusActive {
		t.Fatalf("edit title: %+v %v", updated.Title, err)
	}
	next = updated.Input
	next.FeeBearer = fees.BearerCustomer
	if _, err := f.svc.Update(ctx, pid, l.ID, next); CodeOf(err) != CodeSurchargeForbidden {
		t.Fatalf("live edit skipped publish validation: %v", err)
	}
}

func TestArchivedLinkIsNotEditableAndOnlyDraftsDelete(t *testing.T) {
	f := newFixture(t)
	ctx, pid := context.Background(), merchant.PlatformID
	archived := f.linkIn(StatusArchived)
	if _, err := f.svc.Update(ctx, pid, archived.ID, archived.Input); CodeOf(err) != CodeNotEditable {
		t.Fatalf("update archived: %v", err)
	}
	for _, st := range []Status{StatusActive, StatusPaused, StatusArchived} {
		if err := f.svc.Delete(ctx, pid, f.linkIn(st).ID); CodeOf(err) != CodeNotDeletable {
			t.Fatalf("delete %s: %v", st, err)
		}
	}
	draft := f.create(validInput())
	if err := f.svc.Delete(ctx, pid, draft.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Get(ctx, pid, draft.ID); CodeOf(err) != CodeNotFound {
		t.Fatalf("deleted draft still readable: %v", err)
	}
}

func TestStaleUpdateIsAConflict(t *testing.T) {
	f := newFixture(t)
	ctx, pid := context.Background(), merchant.PlatformID
	l := f.create(validInput())
	if _, err := f.svc.Update(ctx, pid, l.ID, l.Input); err != nil {
		t.Fatal(err)
	}
	stale := l
	stale.Title = "older"
	if _, err := f.store.Save(ctx, stale); err != errStale {
		t.Fatalf("save on an old revision: %v", err)
	}
}

func TestDuplicateMakesAFreshDraft(t *testing.T) {
	f := newFixture(t)
	other := Actor{MemberID: 8, PlatformID: merchant.PlatformID}
	in := validInput()
	in.Questions = []Question{{Key: "size", Label: "Size", Type: QuestionSelect, Options: []string{"S", "M"}}}
	l := f.published(in)
	dup, err := f.svc.Duplicate(context.Background(), other, l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dup.ID == l.ID || dup.Status != StatusDraft || dup.ShortCode != "" || dup.UsesCount != 0 || dup.MemberID != 8 {
		t.Fatalf("duplicate %+v", dup)
	}
	if dup.Title != l.Title || !dup.Amount.Equal(*l.Amount) || dup.Questions[0].Options[1] != "M" {
		t.Fatalf("duplicate lost the form: %+v", dup.Input)
	}
	dup.Questions[0].Options[0] = "XL"
	again, _ := f.svc.Get(context.Background(), merchant.PlatformID, l.ID)
	if again.Questions[0].Options[0] != "S" {
		t.Fatal("duplicate shares memory with its source")
	}
}

func TestLinksAreScopedToTheirPlatform(t *testing.T) {
	f := newFixture(t)
	l := f.create(validInput())
	if _, err := f.svc.Get(context.Background(), 99, l.ID); CodeOf(err) != CodeNotFound {
		t.Fatalf("other platform read the link: %v", err)
	}
	if _, err := f.svc.Publish(context.Background(), 99, l.ID); CodeOf(err) != CodeNotFound {
		t.Fatalf("other platform published the link: %v", err)
	}
	list, total, _ := f.svc.List(context.Background(), 99, ListFilter{})
	if total != 0 || len(list) != 0 {
		t.Fatalf("other platform listed %d", total)
	}
}

func TestListFiltersByStatus(t *testing.T) {
	f := newFixture(t)
	f.create(validInput())
	f.linkIn(StatusActive)
	list, total, err := f.svc.List(context.Background(), merchant.PlatformID, ListFilter{Status: StatusActive})
	if err != nil || total != 1 || list[0].Status != StatusActive {
		t.Fatalf("list %d %v", total, err)
	}
	if _, _, err := f.svc.List(context.Background(), merchant.PlatformID, ListFilter{Status: "gone"}); CodeOf(err) != CodeInvalidRequest {
		t.Fatalf("bad status filter: %v", err)
	}
}
