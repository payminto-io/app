package solana

import "testing"

func TestAssociatedTokenAddress_MatchesSplTokenCLI(t *testing.T) {
	owner := MustPublicKey("HAgk14JpMQLgt6rVgv7cBQFJWFto5Dqxi472uT3DKpqk")
	usdc := MustPublicKey("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
	ata, err := AssociatedTokenAddress(owner, usdc, TokenProgram)
	if err != nil {
		t.Fatal(err)
	}
	// spl-token address --owner HAgk... --token EPjF... --url mainnet-beta
	if got, want := ata.String(), "5N3f1tj9v1vc5TUZ8S7mCAnVmjVKrfnzXWhxLaxyZAgt"; got != want {
		t.Fatalf("ata = %s, want %s", got, want)
	}
	if ata.IsOnCurve() {
		t.Fatal("ata must be off curve")
	}
	if !owner.IsOnCurve() {
		t.Fatal("an ed25519 public key is on the curve")
	}
}

func TestParsePublicKey_RejectsWrongLength(t *testing.T) {
	if _, err := ParsePublicKey("abc"); err == nil {
		t.Fatal("expected error")
	}
}

func TestTokenProgramFor(t *testing.T) {
	if p, _ := TokenProgramFor("SPL"); p != TokenProgram {
		t.Fatal("SPL should map to the token program")
	}
	if p, _ := TokenProgramFor("SPL-2022"); p != Token2022Program {
		t.Fatal("SPL-2022 should map to token-2022")
	}
	if _, err := TokenProgramFor("ERC20"); err == nil {
		t.Fatal("ERC20 is not a Solana standard")
	}
}
