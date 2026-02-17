// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"testing"
)

func TestPasswordHashAndVerify(t *testing.T) {
	password := "correct-horse-battery-staple"

	hashed, err := PasswordHash(password)
	if err != nil {
		t.Fatalf("PasswordHash returned error: %v", err)
	}

	ok, err := PasswordVerify(password, hashed)
	if err != nil {
		t.Fatalf("PasswordVerify returned error: %v", err)
	}
	if !ok {
		t.Error("PasswordVerify should return true for the correct password")
	}

	ok, err = PasswordVerify("wrong-password", hashed)
	if err != nil {
		t.Fatalf("PasswordVerify returned error: %v", err)
	}
	if ok {
		t.Error("PasswordVerify should return false for an incorrect password")
	}
}

func TestPasswordHashUniqueSalts(t *testing.T) {
	password := "same-password"

	hash1, err := PasswordHash(password)
	if err != nil {
		t.Fatalf("PasswordHash returned error: %v", err)
	}
	hash2, err := PasswordHash(password)
	if err != nil {
		t.Fatalf("PasswordHash returned error: %v", err)
	}

	if hash1 == hash2 {
		t.Error("Two hashes of the same password should differ due to random salts")
	}

	for i, h := range []string{hash1, hash2} {
		ok, err := PasswordVerify(password, h)
		if err != nil {
			t.Fatalf("PasswordVerify returned error for hash %d: %v", i+1, err)
		}
		if !ok {
			t.Errorf("PasswordVerify should return true for hash %d", i+1)
		}
	}
}

func TestPasswordVerifyInvalidStoredPassword(t *testing.T) {
	tests := []struct {
		name   string
		stored string
	}{
		{"too short", "0abcde"},
		{"wrong version", "1abcdefghij" + "aa"},
		{"invalid hex", "0abcdefghij" + "zzzz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PasswordVerify("password", tc.stored)
			if err == nil {
				t.Error("PasswordVerify should return an error for invalid stored password")
			}
		})
	}
}
