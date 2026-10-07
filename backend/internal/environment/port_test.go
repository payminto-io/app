package environment

import (
	"context"
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	for in, want := range map[string]Environment{"test": Test, "LIVE": Live, " live ": Live} {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "prod", "sandbox", "staging"} {
		if _, err := Parse(in); !errors.Is(err, ErrInvalid) {
			t.Errorf("Parse(%q) error = %v, want ErrInvalid", in, err)
		}
	}
}

func TestKeyPrefixRoundTrip(t *testing.T) {
	for _, env := range All() {
		for _, kind := range []KeyKind{SecretKey, PublishableKey} {
			got, gotKind, ok := KeyEnvironment(env.KeyPrefix(kind) + "abc")
			if !ok || got != env || gotKind != kind {
				t.Errorf("KeyEnvironment(%s) = %s %s %v", env.KeyPrefix(kind), got, gotKind, ok)
			}
		}
	}
	if _, _, ok := KeyEnvironment("pm_legacy"); ok {
		t.Error("legacy pm_ key must not parse to an environment")
	}
	if Live.KeyPrefix(SecretKey) != "sk_live_" || Test.KeyPrefix(PublishableKey) != "pk_test_" {
		t.Error("prefix format changed")
	}
}

func TestContextRoundTrip(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Error("empty context must not carry an environment")
	}
	env, ok := FromContext(WithContext(context.Background(), Live))
	if !ok || env != Live {
		t.Errorf("FromContext = %q %v", env, ok)
	}
}

func TestMismatchErrorIs(t *testing.T) {
	err := error(&MismatchError{Process: Live, Requested: Test})
	if !errors.Is(err, ErrMismatch) {
		t.Fatal("MismatchError must satisfy errors.Is(ErrMismatch)")
	}
	var typed *MismatchError
	if !errors.As(err, &typed) || typed.Requested != Test {
		t.Fatal("MismatchError must be recoverable with errors.As")
	}
}

func TestDeriveKeyAndAudience(t *testing.T) {
	secret := []byte("shared-secret")
	testKey, err := DeriveKey(secret, Test, "jwt-access")
	if err != nil {
		t.Fatal(err)
	}
	liveKey, _ := DeriveKey(secret, Live, "jwt-access")
	again, _ := DeriveKey(secret, Test, "jwt-access")
	other, _ := DeriveKey(secret, Test, "refresh-token")
	if string(testKey) == string(liveKey) || string(testKey) == string(other) || string(testKey) != string(again) || len(testKey) != 32 {
		t.Fatal("derived keys must differ per environment and purpose and be deterministic")
	}
	if _, err := DeriveKey(secret, "prod", "x"); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid environment accepted")
	}
	if _, err := DeriveKey(nil, Test, "x"); err == nil {
		t.Fatal("empty secret accepted")
	}
	if env, ok := AudienceEnvironment(Live.Audience()); !ok || env != Live {
		t.Fatalf("AudienceEnvironment(%q) = %q %v", Live.Audience(), env, ok)
	}
	if _, ok := AudienceEnvironment("other:live"); ok {
		t.Fatal("foreign audience parsed")
	}
}
