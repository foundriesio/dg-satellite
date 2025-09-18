// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package server

import (
	"encoding/json"
	"fmt"
	"time"
)

type Authenticator struct {
	tokenCipher ServerCipher
}

func NewAuthenticator(tokenSecret string) (a Authenticator, err error) {
	const tokenCipherVersion uint8 = 1
	if a.tokenCipher, err = NewCipher(tokenCipherVersion, tokenSecret); err != nil {
		err = fmt.Errorf("failed to initialize token cipher: %w", err)
	}
	return
}

func (a Authenticator) NewToken(data any, expiresIn time.Duration) (string, error) {
	var (
		t   tokenBody
		err error
	)
	if t.Data, err = json.Marshal(data); err != nil {
		return "", fmt.Errorf("failed marshaling token JSON: %w", err)
	}
	t.Expires = time.Now().Add(expiresIn).Unix()
	if token, err := json.Marshal(data); err != nil {
		return "", fmt.Errorf("failed marshaling token JSON: %w", err)
	} else if enc, err := a.tokenCipher.Encrypt(token); err != nil {
		return "", fmt.Errorf("failed encrypting token: %w", err)
	} else {
		return string(enc), nil
	}
}

func (a Authenticator) ParseToken(data any, token string) error {
	if dec, err := a.tokenCipher.Decrypt([]byte(token)); err != nil {
		return fmt.Errorf("failed decrypting token: %w", err)
	} else {
		var t tokenBody
		if err = json.Unmarshal(dec, &t); err != nil {
			return fmt.Errorf("failed parsing token JSON: %w", err)
		}
		if time.Now().After(time.Unix(t.Expires, 0)) {
			return fmt.Errorf("token expired at unix timestamp: %d", t.Expires)
		}
		if err = json.Unmarshal(t.Data, data); err != nil {
			return fmt.Errorf("failed parsing token JSON: %w", err)
		}
		return nil
	}
}

type tokenBody struct {
	Data    json.RawMessage `json:"data"`
	Expires int64           `json:"expires"`
}
