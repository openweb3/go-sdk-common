package privatekeyhelper

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestPrivateKeyParsingCompatibility(t *testing.T) {
	// Public test key: scalar 1, never use for funds.
	hexKey := strings.Repeat("0", 63) + "1"
	for _, input := range []string{hexKey, "0x" + hexKey} {
		key, err := NewFromKeyString(input)
		if err != nil {
			t.Fatal(err)
		}
		if got := crypto.PubkeyToAddress(key.PublicKey).Hex(); got != "0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf" {
			t.Fatalf("scalar 1 address = %s", got)
		}
	}
	for _, input := range []string{"", "xyz", "01", strings.Repeat("0", 64), strings.Repeat("f", 64)} {
		if _, err := NewFromKeyString(input); err == nil {
			t.Errorf("accepted invalid private key %q", input)
		}
	}
}

func TestMnemonicDerivationCompatibility(t *testing.T) {
	// Public Hardhat test mnemonic and its first two Ethereum accounts.
	const mnemonic = "test test test test test test test test test test test junk"
	for i, want := range []string{"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"} {
		key, err := NewFromMnemonic(mnemonic, i, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := crypto.PubkeyToAddress(key.PublicKey).Hex(); got != want {
			t.Fatalf("account %d address = %s, want %s", i, got, want)
		}
	}
	if _, err := NewFromMnemonic("invalid mnemonic", 0, nil); !errors.Is(err, ErrInvalidMnemonic) {
		t.Fatalf("invalid mnemonic error = %v", err)
	}
}

func TestKeystoreCompatibility(t *testing.T) {
	key, err := NewFromKeyString(strings.Repeat("0", 63) + "1")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := ToKeystore(key, "test-password")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := NewFromKeystore(encoded, "test-password")
	if err != nil {
		t.Fatal(err)
	}
	if decoded.D.Cmp(key.D) != 0 {
		t.Fatal("keystore round trip changed the private key")
	}
	if _, err := NewFromKeystore(encoded, "wrong-password"); err == nil {
		t.Fatal("keystore accepted wrong password")
	}
	if _, err := NewFromKeystore([]byte("invalid JSON"), "test-password"); err == nil {
		t.Fatal("keystore accepted malformed JSON")
	}
}
