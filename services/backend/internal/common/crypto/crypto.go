// Package crypto provides authenticated encryption for secrets stored at rest
// (SSH private keys, registry/DNS API tokens). It is AES-256-GCM under a single
// master key supplied at startup via SHIPPER_API_SECRETS_MASTER_KEY. The Box
// type satisfies the per-domain Secrets ports, so domains never import this
// package directly and can be tested with an in-memory fake.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// KeyLen is the required master-key length: AES-256 needs 32 bytes.
const KeyLen = 32

// Box seals and opens secrets with AES-256-GCM. Each Seal draws a fresh random
// nonce, returned alongside the ciphertext — store both; the nonce is not
// secret. Open verifies the GCM tag, so any tampering (or a wrong key) fails.
//
// Box supports rotation via an optional previous key (NewBoxWithPrev): Seal
// always uses the primary key, while Open first tries the primary and falls
// back to the previous one. Records sealed with the old key continue to open
// until they are next written (lazy re-seal); the operator removes the
// previous key once an audit/migration confirms all rows have been re-sealed.
type Box struct {
	aead     cipher.AEAD
	prevAEAD cipher.AEAD // nil when no rotation key is configured
}

// NewBox builds a Box from a 32-byte key, erroring on a wrong-length key so a
// misconfigured master key fails fast at startup.
func NewBox(key []byte) (*Box, error) {
	return NewBoxWithPrev(key, nil)
}

// NewBoxWithPrev builds a rotation-aware Box. The primary key is used for all
// Seal calls and tried first on Open; prev (when non-nil) is used as the
// fallback decrypt key for records still sealed under the previous master.
// Pass nil for prev to disable rotation. A wrong-length prev key returns an
// error.
func NewBoxWithPrev(primary, prev []byte) (*Box, error) {
	aead, err := makeAEAD(primary)
	if err != nil {
		return nil, err
	}
	var prevAEAD cipher.AEAD
	if len(prev) > 0 {
		prevAEAD, err = makeAEAD(prev)
		if err != nil {
			return nil, err
		}
	}
	return &Box{aead: aead, prevAEAD: prevAEAD}, nil
}

func makeAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != KeyLen {
		return nil, errors.New("crypto: master key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Seal encrypts plaintext, returning the ciphertext and the nonce used. Persist
// both; Open needs the nonce.
func (b *Box) Seal(plaintext []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, b.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return b.aead.Seal(nil, nonce, plaintext, nil), nonce, nil
}

// Open decrypts ciphertext sealed with the same key and nonce. When a previous
// key is configured (NewBoxWithPrev) and primary-key decryption fails, Open
// retries under the previous key — supporting lazy re-seal during rotation.
// Returns an error if neither key authenticates the payload; the error is not
// wrapped, to avoid leaking cryptographic detail.
func (b *Box) Open(ciphertext, nonce []byte) ([]byte, error) {
	plain, err := b.aead.Open(nil, nonce, ciphertext, nil)
	if err == nil {
		return plain, nil
	}
	if b.prevAEAD == nil {
		return nil, err
	}
	return b.prevAEAD.Open(nil, nonce, ciphertext, nil)
}
