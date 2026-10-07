package none

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/cre/conformance"
)

func TestConformance(t *testing.T) {
	conformance.Run(t, conformance.Harness{Attester: New(), ExpectTriggerErr: cre.ErrDisabled})
}
