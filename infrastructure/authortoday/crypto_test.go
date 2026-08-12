package authortoday

import (
	"os"
	"strings"
	"testing"
)

// secretFromFixture is the per-request Reader-Secret header value captured from
// https://author.today/reader/15077/chapter?id=86697 for the chapter "Пролог" of
// "Книга первая - Меняя маски". The ciphertext it protects is stored alongside
// this test (testdata/prologue.enc). author.today rotates the secret per
// request, but for any fixed (secret, ciphertext) pair DecodeText is
// deterministic, so this is a faithful regression guard for the exact
// decryption scheme: key = reverse(secret) + "@_@"  (userId is empty, NOT the
// numeric account id — appending it corrupts the key and produces garbage).
const secretFromFixture = "3634a48232210d9981a46c6d4f0776e2"

func TestDecodeText_RealVector(t *testing.T) {
	raw, err := os.ReadFile("testdata/prologue.enc")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	decoded := DecodeText(secretFromFixture, string(raw))

	if !strings.HasPrefix(decoded, "<p>") {
		t.Fatalf("expected decrypted text to start with '<p>', got: %q", decoded[:40])
	}
	if strings.Count(decoded, "<p>") != strings.Count(decoded, "</p>") {
		t.Fatalf("unbalanced <p> tags: open=%d close=%d", strings.Count(decoded, "<p>"), strings.Count(decoded, "</p>"))
	}
	if !strings.Contains(decoded, "Приветствую") {
		t.Fatalf("expected readable Russian in decrypted text, got: %q", decoded[:120])
	}
	bad := 0
	for _, r := range decoded {
		o := int(r)
		if o < 0x20 && o != '\n' && o != '\t' && o != '\r' {
			bad++
		}
	}
	if bad > 0 {
		t.Fatalf("decrypted text contains %d control chars — decryption is wrong", bad)
	}
}

func TestDecodeText_Roundtrip(t *testing.T) {
	secret := "deadbeef"
	original := "Hello, author.today! Привет, мир. <p>Тест</p>"
	encoded := DecodeText(secret, original)
	decoded := DecodeText(secret, encoded)
	if decoded != original {
		t.Fatalf("roundtrip failed: got %q want %q", decoded, original)
	}
}

func TestDecodeText_UsesReversedSecretAndNoUserID(t *testing.T) {
	// Locks the exact key construction against the regression where the numeric
	// user id was appended (which yields garbage). The key must be
	// reverse(secret) + "@_@" only — appending a user id changes the key.
	secret := "abcd1234"
	plaintext := "<p>Тест короткий текст для проверки ключа</p>"
	pr := []rune(plaintext)
	key := append(reverseRunes([]rune(secret)), []rune("@_@")...)
	encrypted := make([]rune, len(pr))
	for i, r := range pr {
		encrypted[i] = r ^ key[i%len(key)]
	}
	if got := DecodeText(secret, string(encrypted)); got != plaintext {
		t.Fatalf("DecodeText(secret, enc) = %q, want %q", got, plaintext)
	}
}

func TestDecodeText_XORMatchesReference(t *testing.T) {
	secret := "k"
	text := "A"
	// reverse("k")+"@_" then "k" -> 'k'? no: reverseRunes("k")=("k"); key="k@_@"
	// first char: 'A'(0x41) ^ key[0]('k'=0x6b) = 0x2a
	secretRunes := append(reverseRunes([]rune(secret)), []rune("@_@")...)
	want := uint32('A') ^ uint32(secretRunes[0%len(secretRunes)])
	got := DecodeText(secret, text)
	if got != string(rune(want)) {
		t.Fatalf("expected %q got %q", string(rune(want)), got)
	}
}

func TestDecodeText_ReversedKey(t *testing.T) {
	if string(reverseRunes([]rune("abc"))) != "cba" {
		t.Fatal("reverseRunes failed")
	}
}
