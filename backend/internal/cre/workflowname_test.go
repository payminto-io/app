package cre

import (
	"encoding/hex"
	"testing"
)

func TestKeystoneNameMatchesTheDocsVector(t *testing.T) {
	got := KeystoneName("my_workflow")
	if hex.EncodeToString(got[:]) != "62373666336165316465" {
		t.Fatalf("my_workflow -> %x", got)
	}
	sol := KeystoneName("solvency")
	if string(sol[:]) != "58c66935b7" {
		t.Fatalf("solvency -> %s", sol[:])
	}
}
