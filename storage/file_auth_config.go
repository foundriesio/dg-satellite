// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"path/filepath"
)

type AuthFsHandle struct {
	baseFsHandle
}

type AuthConfig struct {
	Type                 string
	SessionTimeoutHours  int // Default is 48 hours
	NewUserDefaultScopes []string
	Config               json.RawMessage
}

func (h AuthFsHandle) InitHmacSecret() error {
	if _, err := h.readFile(HmacFile, false); err == nil {
		path := filepath.Join(h.root, HmacFile)
		return fmt.Errorf("hmac secret exists at: %s", path)
	}

	secret := make([]byte, 64)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("generating HMAC secret: %w", err)
	}
	if err := h.writeFile(HmacFile, string(secret), 0o640); err != nil {
		return fmt.Errorf("storing HMAC secret: %w", err)
	}
	return nil
}

func (h AuthFsHandle) GetHmacSecret() ([]byte, error) {
	secret, err := h.readFile(HmacFile, false)
	return []byte(secret), err
}

// GetAuthConfig returns the settings for how authorization is configured.
func (h AuthFsHandle) GetAuthConfig() (*AuthConfig, error) {
	var cfg AuthConfig
	handle := baseFsHandle{root: h.root}
	contents, err := handle.readFile(AuthConfigFile, false)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(contents), &cfg); err != nil {
		return nil, fmt.Errorf("unable to unmarshall auth config: %w", err)
	}
	if cfg.SessionTimeoutHours == 0 {
		cfg.SessionTimeoutHours = 48
	}
	return &cfg, nil
}

func (h AuthFsHandle) SaveAuthConfig(cfg AuthConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshall auth config: %w", err)
	}
	if err := h.writeFile(AuthConfigFile, string(data), 0o640); err != nil {
		return fmt.Errorf("storing auth config: %w", err)
	}
	return nil
}
