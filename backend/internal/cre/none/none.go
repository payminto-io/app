// Package none is the default provider: the module is wired, nothing runs, every call says so.
package none

import (
	"context"

	"github.com/payminto/payminto/backend/internal/cre"
)

type Attester struct{}

var _ cre.Attester = Attester{}

func New() Attester { return Attester{} }

func (Attester) Name() string { return cre.ProviderNone }

func (Attester) Trigger(context.Context, cre.Kind, []byte) (string, error) {
	return "", cre.ErrDisabled
}

func (Attester) Poll(_ context.Context, _ cre.Kind, cursor cre.Cursor) ([]cre.RawAttestation, cre.Cursor, error) {
	return nil, cursor, nil
}

func (Attester) Health(context.Context) cre.Health {
	return cre.Health{Status: cre.HealthOff, Message: "attestations are off (CRE_ENABLED=false)"}
}
