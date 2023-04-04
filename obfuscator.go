package caching

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

const (
	keyBytes = 32
)

// Obfuscator struct to hold random key bytes
type Obfuscator struct {
	key []byte
}

// NewObfuscator generates a random 256-bit key for obfuscation
func NewObfuscator() *Obfuscator {
	buf := make([]byte, keyBytes)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}

	return &Obfuscator{
		key: buf,
	}
}

// Obfuscate method obfuscate data using 256-bit AES-GCM. This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
func (obfuscator *Obfuscator) Obfuscate(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(obfuscator.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Deobfuscate method deobfuscate the data using 256-bit AES-GCM. This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
func (obfuscator *Obfuscator) Deobfuscate(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(obfuscator.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
}
