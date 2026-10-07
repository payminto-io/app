package links

import (
	"context"
	"maps"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MemStore is an in-memory Store for tests and examples; one mutex makes Reserve atomic.
type MemStore struct {
	mu        sync.Mutex
	links     map[string]Link
	payments  map[string]LinkPayment
	webhooks  map[uint]uint
	merchants map[uint]string
	// failComplete, when set, is returned once by Complete (FailNextComplete).
	failComplete error
}

func NewMemStore() *MemStore {
	return &MemStore{links: map[string]Link{}, payments: map[string]LinkPayment{}, webhooks: map[uint]uint{}, merchants: map[uint]string{}}
}

// AddWebhook registers a webhook id as belonging to a platform.
func (s *MemStore) AddWebhook(platformID, webhookID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhooks[webhookID] = platformID
}

func (s *MemStore) SetMerchantName(platformID uint, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.merchants[platformID] = name
}

func cloneDec(d *decimal.Decimal) *decimal.Decimal {
	if d == nil {
		return nil
	}
	v := *d
	return &v
}

func cloneInt(v *int) *int {
	if v == nil {
		return nil
	}
	n := *v
	return &n
}

func cloneLink(l Link) Link {
	l.Amount, l.AmountMin, l.AmountMax, l.Total = cloneDec(l.Amount), cloneDec(l.AmountMin), cloneDec(l.AmountMax), cloneDec(l.Total)
	l.Metadata = maps.Clone(l.Metadata)
	l.UseLimit, l.ExpiresAfterPayments = cloneInt(l.UseLimit), cloneInt(l.ExpiresAfterPayments)
	l.Methods = slices.Clone(l.Methods)
	l.LineItems = slices.Clone(l.LineItems)
	qs := make([]Question, len(l.Questions))
	for i, q := range l.Questions {
		q.Options = slices.Clone(q.Options)
		qs[i] = q
	}
	l.Questions = qs
	if l.WebhookID != nil {
		w := *l.WebhookID
		l.WebhookID = &w
	}
	if l.SettlementOverride != nil {
		o := *l.SettlementOverride
		l.SettlementOverride = &o
	}
	if l.ExpiresAt != nil {
		t := *l.ExpiresAt
		l.ExpiresAt = &t
	}
	if l.PublishedAt != nil {
		t := *l.PublishedAt
		l.PublishedAt = &t
	}
	return l
}

func (s *MemStore) Insert(_ context.Context, l Link) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	l.ID, l.Revision, l.CreatedAt, l.UpdatedAt = uuid.NewString(), 1, now, now
	s.links[l.ID] = cloneLink(l)
	return cloneLink(l), nil
}

func (s *MemStore) Get(_ context.Context, platformID uint, id string) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.links[id]
	if !ok || l.ExternalPlatformID != platformID {
		return Link{}, ErrStoreNotFound
	}
	return cloneLink(l), nil
}

func (s *MemStore) GetByShortCode(_ context.Context, code string) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, l := range s.links {
		if l.ShortCode == code {
			return cloneLink(l), nil
		}
	}
	return Link{}, ErrStoreNotFound
}

func (s *MemStore) List(_ context.Context, platformID uint, f ListFilter) ([]Link, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []Link
	for _, l := range s.links {
		if l.ExternalPlatformID == platformID && (f.Status == "" || l.Status == f.Status) {
			all = append(all, cloneLink(l))
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].ID > all[j].ID
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	total := int64(len(all))
	start := min(f.Offset, len(all))
	end := min(start+f.Limit, len(all))
	return all[start:end], total, nil
}

func (s *MemStore) Save(_ context.Context, l Link) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.links[l.ID]
	if !ok || cur.ExternalPlatformID != l.ExternalPlatformID {
		return Link{}, ErrStoreNotFound
	}
	if cur.Revision != l.Revision {
		return Link{}, ErrStale
	}
	next := cloneLink(l)
	cur.Input, cur.Total = next.Input, next.Total
	cur.Revision++
	cur.UpdatedAt = time.Now().UTC()
	s.links[l.ID] = cur
	return cloneLink(cur), nil
}

func (s *MemStore) SetStatus(_ context.Context, platformID uint, id string, revision int, to Status, code string, now time.Time) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.links[id]
	if !ok || cur.ExternalPlatformID != platformID {
		return Link{}, ErrStoreNotFound
	}
	if cur.Revision != revision {
		return Link{}, ErrStale
	}
	if code != "" && cur.ShortCode == "" {
		for _, other := range s.links {
			if other.ShortCode == code {
				return Link{}, ErrShortCodeTaken
			}
		}
		cur.ShortCode = code
	}
	if to == StatusActive && cur.PublishedAt == nil {
		t := now
		cur.PublishedAt = &t
	}
	cur.Status = to
	cur.Revision++
	cur.UpdatedAt = now
	s.links[id] = cur
	return cloneLink(cur), nil
}

func (s *MemStore) DeleteDraft(_ context.Context, platformID uint, id string, revision int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.links[id]
	if !ok || cur.ExternalPlatformID != platformID {
		return ErrStoreNotFound
	}
	if cur.Revision != revision || cur.Status != StatusDraft {
		return ErrStale
	}
	delete(s.links, id)
	return nil
}

func (s *MemStore) WebhookExists(_ context.Context, platformID, webhookID uint) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.webhooks[webhookID]
	return ok && p == platformID, nil
}

func (s *MemStore) MerchantName(_ context.Context, platformID uint) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.merchants[platformID], nil
}

func (s *MemStore) findLocked(linkID, key string) *LinkPayment {
	for _, p := range s.payments {
		if p.LinkID == linkID && p.IdempotencyKey == key && p.Status != paymentReleased {
			c := p
			return &c
		}
	}
	return nil
}

func (s *MemStore) FindPayment(_ context.Context, linkID, key string) (*LinkPayment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.findLocked(linkID, key), nil
}

func (s *MemStore) Reserve(_ context.Context, p LinkPayment, now time.Time, limits ReserveLimits) (LinkPayment, *LinkPayment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.findLocked(p.LinkID, p.IdempotencyKey); existing != nil {
		return LinkPayment{}, existing, nil
	}
	l, ok := s.links[p.LinkID]
	if !ok {
		return LinkPayment{}, nil, ErrStoreNotFound
	}
	if err := availability(l, now); err != nil {
		return LinkPayment{}, nil, err
	}
	if l.FeeBearer != p.FeeBearer {
		return LinkPayment{}, nil, ErrStale
	}
	open, mine := 0, 0
	for _, q := range s.payments {
		if q.LinkID == p.LinkID && q.Status != paymentReleased && q.OpenUntil.After(now) {
			open++
			if q.ClientKey == p.ClientKey {
				mine++
			}
		}
	}
	if err := openPaymentsError(l, limits, open, mine); err != nil {
		return LinkPayment{}, nil, err
	}
	l.UsesCount++
	s.links[l.ID] = l
	p.ID, p.Status, p.CreatedAt = uuid.NewString(), paymentPending, now
	s.payments[p.ID] = p
	return p, nil, nil
}

func (s *MemStore) Complete(_ context.Context, id string, created CreatedPayment, openUntil time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failComplete; err != nil {
		s.failComplete = nil
		return err
	}
	p, ok := s.payments[id]
	if !ok || p.Status != paymentPending {
		return ErrStale
	}
	p.Status, p.PaymentReference, p.Processor, p.OpenUntil = paymentCreated, created.Reference, &created, openUntil
	s.payments[id] = p
	return nil
}

func (s *MemStore) Release(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.payments[id]
	if !ok || p.Status != paymentPending {
		return ErrStale
	}
	p.Status = paymentReleased
	s.payments[id] = p
	if l, ok := s.links[p.LinkID]; ok {
		l.UsesCount--
		s.links[l.ID] = l
	}
	return nil
}

func (s *MemStore) ExpiredPending(_ context.Context, now time.Time, limit int) ([]LinkPayment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []LinkPayment
	for _, p := range s.payments {
		if p.Status == paymentPending && p.ReservedUntil.Before(now) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ReservedUntil.Before(out[j].ReservedUntil) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemStore) Link(_ context.Context, id string) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.links[id]
	if !ok {
		return Link{}, ErrStoreNotFound
	}
	return cloneLink(l), nil
}

func (s *MemStore) OpenCreated(_ context.Context, linkID string, now time.Time, limit int) ([]LinkPayment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []LinkPayment
	for _, p := range s.payments {
		if p.LinkID == linkID && p.Status == paymentCreated && p.OpenUntil.After(now) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemStore) CloseUses(_ context.Context, ids []string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		if p, ok := s.payments[id]; ok && p.Status == paymentCreated && p.OpenUntil.After(now) {
			p.OpenUntil = now
			s.payments[id] = p
		}
	}
	return nil
}

// FailNextComplete makes the next Complete return err without writing, to test the window after creation.
func (s *MemStore) FailNextComplete(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failComplete = err
}

// Payments returns every stored use of a link that is not released, for tests.
func (s *MemStore) Payments(linkID string) []LinkPayment {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []LinkPayment
	for _, p := range s.payments {
		if p.LinkID == linkID && p.Status != paymentReleased {
			out = append(out, p)
		}
	}
	return out
}
