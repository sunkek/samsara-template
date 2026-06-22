package crypto

import (
	"bytes"
	"testing"
)

func key(b byte) []byte {
	k := make([]byte, KeyLen)
	for i := range k {
		k[i] = b
	}
	return k
}

func TestNewBoxKeyLength(t *testing.T) {
	for _, tc := range []struct {
		name string
		len  int
		ok   bool
	}{
		{"too short", 16, false},
		{"too long", 64, false},
		{"exact", KeyLen, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewBox(make([]byte, tc.len))
			if tc.ok && err != nil {
				t.Fatalf("want ok, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}

func TestSealOpenRoundTrip(t *testing.T) {
	b, err := NewBox(key(0x01))
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n")
	ct, nonce, err := b.Seal(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ct, plain) {
		t.Fatal("ciphertext equals plaintext")
	}
	got, err := b.Open(ct, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("round-trip mismatch: %q", got)
	}
}

func TestSealUsesFreshNonce(t *testing.T) {
	b, _ := NewBox(key(0x02))
	_, n1, _ := b.Seal([]byte("x"))
	_, n2, _ := b.Seal([]byte("x"))
	if bytes.Equal(n1, n2) {
		t.Fatal("nonce reused across Seal calls")
	}
}

func TestOpenWrongKeyFails(t *testing.T) {
	a, _ := NewBox(key(0x03))
	other, _ := NewBox(key(0x04))
	ct, nonce, _ := a.Seal([]byte("data"))
	if _, err := other.Open(ct, nonce); err == nil {
		t.Fatal("open with wrong key should fail")
	}
}

func TestPrevKeyOpensLegacyRecords(t *testing.T) {
	// Records sealed under the old master must keep opening after rotation.
	old, _ := NewBox(key(0x10))
	ct, nonce, _ := old.Seal([]byte("legacy"))

	// Rotated Box: new primary + old as previous.
	rotated, err := NewBoxWithPrev(key(0x11), key(0x10))
	if err != nil {
		t.Fatal(err)
	}
	got, err := rotated.Open(ct, nonce)
	if err != nil {
		t.Fatalf("rotated Box failed to open legacy record: %v", err)
	}
	if !bytes.Equal(got, []byte("legacy")) {
		t.Fatalf("plaintext mismatch: %q", got)
	}
}

func TestPrevKeyNotUsedForSeal(t *testing.T) {
	// Seal must use the primary key only. Sealing under rotated Box and opening
	// with a Box that only has the primary must succeed; opening with a Box
	// that only has the old key must fail.
	rotated, _ := NewBoxWithPrev(key(0x21), key(0x20))
	ct, nonce, _ := rotated.Seal([]byte("fresh"))

	primaryOnly, _ := NewBox(key(0x21))
	if _, err := primaryOnly.Open(ct, nonce); err != nil {
		t.Fatalf("primary-only Box should open freshly sealed data: %v", err)
	}
	oldOnly, _ := NewBox(key(0x20))
	if _, err := oldOnly.Open(ct, nonce); err == nil {
		t.Fatal("fresh seal must not be openable with the previous-only key")
	}
}

func TestPrevKeyOpenFailsForUnknownKey(t *testing.T) {
	rotated, _ := NewBoxWithPrev(key(0x31), key(0x30))
	stranger, _ := NewBox(key(0x99))
	ct, nonce, _ := stranger.Seal([]byte("alien"))
	if _, err := rotated.Open(ct, nonce); err == nil {
		t.Fatal("rotated Box must reject data sealed under a third unrelated key")
	}
}

func TestOpenTamperedFails(t *testing.T) {
	b, _ := NewBox(key(0x05))
	ct, nonce, _ := b.Seal([]byte("data"))
	ct[0] ^= 0xff
	if _, err := b.Open(ct, nonce); err == nil {
		t.Fatal("open of tampered ciphertext should fail")
	}
}
