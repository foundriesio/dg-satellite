// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/scrypt"
)

func PasswordHash(password string) (string, error) {
	salt := rand.Text()[:10]
	dk, err := scrypt.Key([]byte(password), []byte(salt), 32768, 8, 1, 32)
	if err != nil {
		return "", fmt.Errorf("unexpected error hashing password: %w", err)
	}
	// Prefix hash with a version number to allow for future changes to the hashing scheme.
	return "0" + salt + hex.EncodeToString(dk), nil
}

func PasswordVerify(password, storedPassword string) (bool, error) {
	if len(storedPassword) < 11 {
		return false, fmt.Errorf("invalid stored password length: %d", len(storedPassword))
	}
	if storedPassword[0] != '0' {
		return false, fmt.Errorf("unsupported password hash version: %c", storedPassword[0])
	}
	salt := []byte(storedPassword[1:11])
	storedHash, err := hex.DecodeString(storedPassword[11:])
	if err != nil {
		return false, fmt.Errorf("unexpected error decoding password hash: %w", err)
	}
	dk, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return false, fmt.Errorf("unexpected error deriving key from password: %w", err)
	}
	return subtle.ConstantTimeCompare(dk, storedHash) == 1, nil
}
