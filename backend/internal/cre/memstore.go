package cre

import (
	"context"
	"sort"
	"sync"
)

// MemoryStore is the in-process Store for unit tests and the conformance suite.
type MemoryStore struct {
	mu       sync.Mutex
	rows     []Attestation
	subjects map[Kind]map[[32]byte]Subject
	runs     []Run
	cursors  map[string]Cursor
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{subjects: map[Kind]map[[32]byte]Subject{}, cursors: map[string]Cursor{}}
}

var _ Store = (*MemoryStore)(nil)

func (m *MemoryStore) SaveAttestations(_ context.Context, rows []Attestation) ([]Attestation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var inserted []Attestation
	for _, r := range rows {
		dup := false
		for _, have := range m.rows {
			if string(have.PayloadHash) == string(r.PayloadHash) && have.ItemIndex == r.ItemIndex {
				dup = true
				break
			}
		}
		if !dup {
			r.Item = ItemJSON(r.Item)
			m.rows = append(m.rows, r)
			inserted = append(inserted, r)
		}
	}
	return inserted, nil
}

func (m *MemoryStore) Seen(_ context.Context, provider string, payloadHash []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.Provider == provider && r.SubjectType != "report" && string(r.PayloadHash) == string(payloadHash) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryStore) sorted(provider string, kind Kind, status Status) []Attestation {
	out := make([]Attestation, 0, len(m.rows))
	for _, r := range m.rows {
		if (kind == "" || r.Kind == kind) && (provider == "" || r.Provider == provider) && (status == "" || r.Status == status) {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].RecordedAt.Equal(out[j].RecordedAt) {
			return out[i].RecordedAt.After(out[j].RecordedAt)
		}
		return out[i].ItemIndex < out[j].ItemIndex
	})
	return out
}

func (m *MemoryStore) ListAttestations(_ context.Context, provider string, kind Kind, limit int) ([]Attestation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.sorted(provider, kind, "")
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryStore) GetAttestation(_ context.Context, id string) (Attestation, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.ID == id {
			return r, true, nil
		}
	}
	return Attestation{}, false, nil
}

func (m *MemoryStore) LatestAttestation(_ context.Context, provider string, kind Kind, status Status) (Attestation, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.sorted(provider, kind, status)
	if len(out) == 0 {
		return Attestation{}, false, nil
	}
	return out[0], true, nil
}

func (m *MemoryStore) RememberSubjects(_ context.Context, subjects []Subject) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range subjects {
		if m.subjects[s.Kind] == nil {
			m.subjects[s.Kind] = map[[32]byte]Subject{}
		}
		if have, ok := m.subjects[s.Kind][s.Key]; ok {
			have.AskedAt = s.AskedAt
			m.subjects[s.Kind][s.Key] = have
			continue
		}
		m.subjects[s.Kind][s.Key] = s
	}
	return nil
}

func (m *MemoryStore) LookupSubject(_ context.Context, kind Kind, key [32]byte) (Subject, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.subjects[kind][key]
	return s, ok, nil
}

func (m *MemoryStore) LatestSubject(_ context.Context, kind Kind) (Subject, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var latest Subject
	found := false
	for _, s := range m.subjects[kind] {
		if !found || s.AskedAt.After(latest.AskedAt) {
			latest, found = s, true
		}
	}
	return latest, found, nil
}

func (m *MemoryStore) RecordRun(_ context.Context, run Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs = append(m.runs, run)
	return nil
}

func (m *MemoryStore) LatestRun(_ context.Context, kind Kind) (Run, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.runs) - 1; i >= 0; i-- {
		if m.runs[i].Kind == kind {
			return m.runs[i], true, nil
		}
	}
	return Run{}, false, nil
}

func (m *MemoryStore) GetCursor(_ context.Context, scope CursorScope) (Cursor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cursors[scope.String()], nil
}

func (m *MemoryStore) SetCursor(_ context.Context, scope CursorScope, c Cursor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cursors[scope.String()] = c
	return nil
}
