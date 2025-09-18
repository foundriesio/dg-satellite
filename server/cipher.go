// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear
package server

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

type ServerCipher interface {
	Decrypt(data []byte) ([]byte, error)
	Encrypt(data []byte) ([]byte, error)
	init(secret, salt []byte) error
}

func NewCipher(version uint8, secret string) (c ServerCipher, err error) {
	if c, err = newCipher(version); err == nil {
		err = c.init([]byte(secret), nil)
	}
	return
}

var ErrCipherVersionMismatch = errors.New("encrypted data version does not match a cipher version")

func newCipher(version uint8) (c ServerCipher, err error) {
	switch version {
	case 1:
		c = newV1Cipher()
	default:
		err = fmt.Errorf("unsupported cipher version: %d", version)
	}
	return
}

type v1CipherAesGcm struct {
	version   uint8
	keySize   int
	saltSize  int
	nonceSize int

	secret []byte
	salt   []byte
	key    []byte
	enc    *base32.Encoding
}

func newV1Cipher() *v1CipherAesGcm {
	return &v1CipherAesGcm{
		version:   1,
		keySize:   32, // For AES 128
		saltSize:  8,  // For Scrypt key derivation salt
		nonceSize: 12, // For GCM random nonce
		enc:       base32.HexEncoding.WithPadding(base32.NoPadding),
	}
}

func (c *v1CipherAesGcm) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cipher block: %w", err)
	}
	mode, err := cipher.NewGCMWithNonceSize(block, c.nonceSize)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cipher mode: %w", err)
	}
	headerSize := 1 + c.saltSize + mode.NonceSize()
	out := make([]byte, headerSize, headerSize+len(data)+mode.Overhead())
	out[0] = c.version
	copy(out[1:1+c.saltSize], c.salt)
	nonce := out[1+c.saltSize : headerSize]
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to initialize cipher nonce: %w", err)
	}
	out = mode.Seal(out, nonce, data, nil)
	wrap := make([]byte, c.enc.EncodedLen(len(out)))
	c.enc.Encode(wrap, out)
	return wrap, nil
}

func (c *v1CipherAesGcm) Decrypt(data []byte) ([]byte, error) {
	var err error
	in := make([]byte, c.enc.DecodedLen(len(data)))
	if _, err = c.enc.Decode(in, data); err != nil {
		return nil, fmt.Errorf("invalid cipher data: failed to parse base32 encoding: %w", err)
	}

	if len(in) < 1 {
		return nil, fmt.Errorf("invalid cipher data: empty stream")
	}
	version := in[0]
	if version != c.version {
		// Different version means a different data parsing.
		// In general use case we should not proceed at this level.
		return nil, fmt.Errorf("%w: cipher version %d, data version %d", ErrCipherVersionMismatch, c.version, version)
	}

	if len(in) < 1+c.saltSize+c.nonceSize {
		return nil, fmt.Errorf("invalid cipher data: insufficient length: %d", len(in))
	}
	if salt := in[1 : 1+c.saltSize]; !bytes.Equal(c.salt, salt) {
		// Same version but different salt - init a new instance and continue.
		c = newV1Cipher()
		if err = c.init(c.secret, salt); err != nil {
			return nil, err
		}
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cipher block: %w", err)
	}
	mode, err := cipher.NewGCMWithNonceSize(block, c.nonceSize)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cipher mode: %w", err)
	}
	headerSize := 1 + c.saltSize + mode.NonceSize()
	nonce := in[1+c.saltSize : headerSize]
	res, err := mode.Open(nil, nonce, in[headerSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("invalid cipher data: failed to decrypt: %w", err)
	}
	return res, nil
}

func (c *v1CipherAesGcm) init(secret, salt []byte) (err error) {
	c.secret = secret
	c.salt = salt
	if len(salt) == 0 {
		c.salt = make([]byte, c.saltSize)
		if _, err = io.ReadFull(rand.Reader, c.salt); err != nil {
			err = fmt.Errorf("failed to generate cipher salt: %w", err)
			return
		}
	}
	if c.key, err = c.deriveKey(c.secret, c.salt); err != nil {
		err = fmt.Errorf("failed to initialize cipher key: %w", err)
	}
	return
}

func (c *v1CipherAesGcm) deriveKey(secret, salt []byte) ([]byte, error) {
	// scrypt parameters: secret, key, N, r, p, keyLen.
	// memory usage = N * r * 128 -> our usage is 16KiB
	// cpu usage ~ N * r * p -> p recommended at 1.
	return scrypt.Key(secret, salt, 16, 8, 1, c.keySize)
}
