package cre

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"gorm.io/gorm"
)

//go:embed schema.sql
var schemaSQL string

// SchemaSQL is the DDL; migration 2026100707_cre_attestations repeats it verbatim (schema_test.go).
func SchemaSQL() string { return schemaSQL }

// Migrate installs the tables for development and test databases; production applies the migration.
func Migrate(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	if err := db.Exec(schemaSQL).Error; err != nil {
		return fmt.Errorf("cre: migrate: %w", err)
	}
	return nil
}

// Run is one trigger of a workflow.
type Run struct {
	Kind        Kind
	Provider    string
	ExecutionID string
	Status      string
	Detail      string
	StartedAt   time.Time
}

const (
	RunAccepted = "accepted"
	RunFailed   = "failed"
)

// Store is what the service persists through; PostgresStore is the implementation, MemoryStore the test double.
type Store interface {
	SubjectIndex
	// SaveAttestations inserts rows and returns only those actually inserted (a concurrent duplicate inserts none).
	SaveAttestations(ctx context.Context, rows []Attestation) ([]Attestation, error)
	// Seen is scoped by provider so rows from a previous provider never block the active one; refusal rows
	// (subject_type "report") never count, so a report refused by configuration can still be recorded later.
	Seen(ctx context.Context, provider string, payloadHash []byte) (bool, error)
	ListAttestations(ctx context.Context, provider string, kind Kind, limit int) ([]Attestation, error)
	GetAttestation(ctx context.Context, id string) (Attestation, bool, error)
	// LatestAttestation is the newest row for the provider and kind; status "" means any status.
	LatestAttestation(ctx context.Context, provider string, kind Kind, status Status) (Attestation, bool, error)
	LatestSubject(ctx context.Context, kind Kind) (Subject, bool, error)
	RecordRun(ctx context.Context, run Run) error
	LatestRun(ctx context.Context, kind Kind) (Run, bool, error)
	GetCursor(ctx context.Context, scope CursorScope) (Cursor, error)
	SetCursor(ctx context.Context, scope CursorScope, c Cursor) error
}
