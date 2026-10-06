package repository

import "testing"

func TestSha256Sum_Deterministic(t *testing.T) {
	s := "pm_abc123"
	h1 := Sha256Sum(s)
	h2 := Sha256Sum(s)
	if h1 != h2 {
		t.Errorf("same input should yield same hash, got %q vs %q", h1, h2)
	}
}

func TestSha256Sum_DifferentInputs(t *testing.T) {
	if Sha256Sum("a") == Sha256Sum("b") {
		t.Error("different inputs should yield different hashes")
	}
}

func TestSha256Sum_HexLength(t *testing.T) {
	h := Sha256Sum("anything")
	if len(h) != 64 {
		t.Errorf("expected 64-char hex, got %d", len(h))
	}
}

func TestSha256Sum_KnownVector(t *testing.T) {
	// SHA-256 of empty string
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := Sha256Sum(""); got != want {
		t.Errorf("SHA-256('') = %q, want %q", got, want)
	}
}
