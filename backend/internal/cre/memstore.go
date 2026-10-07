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
	cursors  map[Kind]Cursor
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{subjects: map[Kind]map[[32]byte]Subject{}, cursors: map[Kind]Cursor{}}
}

var _ Store = (*MemoryStore)(nil)

func (m *MemoryStore) SaveAttestations(_ context.Context, rows []Attestation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range rows {
		dup := false
		for _, have := range m.rows {
			if string(have.PayloadHash) == string(r.PayloadHash) && have.SubjectID == r.SubjectID {
				dup = true
				break
			}
		}
		if !dup {
			r.Item = ItemJSON(r.Item)
			m.rows = append(m.rows, r)
		}
	}
	return nil
}

func (m *MemoryStore) Seen(_ context.Context, payloadHash []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if string(r.PayloadHash) == string(payloadHash) {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryStore) sorted(kind Kind) []Attestation {
	out := make([]Attestation, 0, len(m.rows))
	for _, r := range m.rows {
		if kind == "" || r.Kind == kind {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].RecordedAt.After(out[j].RecordedAt) })
	return out
}

func (m *MemoryStore) ListAttestations(_ context.Context, kind Kind, limit int) ([]Attestation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.sorted(kind)
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

func (m *MemoryStore) LatestAttestation(_ context.Context, kind Kind) (Attestation, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.sorted(kind)
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

func (m *MemoryStore) GetCursor(_ context.Context, kind Kind) (Cursor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cursors[kind], nil
}

func (m *MemoryStore) SetCursor(_ context.Context, kind Kind, c Cursor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cursors[kind] = c
	return nil
}
