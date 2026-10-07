package ledger

import (
	"context"
	"testing"
)

func TestReconcile_ReportsOnlyMismatches(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	mustPost(t, s, balancedJournal())

	expected := []Expected{
		{Account: key(OwnerMember, "m1", "USDC", KindLiability), Stored: dec("10")},
		{Account: key(OwnerPlatform, "hot", "USDC", KindAsset), Stored: dec("10")},
		{Account: key(OwnerMember, "m1", "SOL", KindLiability), Stored: dec("2")},
		{Account: key(OwnerMember, "m2", "USDC", KindLiability), Stored: dec("0")},
	}
	drifts, err := s.Reconcile(ctx, expected)
	if err != nil {
		t.Fatal(err)
	}
	if len(drifts) != 1 || drifts[0].Account.Asset != "SOL" || !drifts[0].Delta().Equal(dec("2")) {
		t.Fatalf("drifts = %+v, want only the unposted SOL balance", drifts)
	}
}
