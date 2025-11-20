// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type AuthConfig struct {
	Type                 string
	NewUserDefaultScopes []string
	Config               json.RawMessage
}

// GetAuthConfig returns the settings for how authorization is configured.
// If no configuration is in place, AuthConfig.Type == ""
func (h FsHandle) GetAuthConfig() (*AuthConfig, error) {
	var cfg AuthConfig
	handle := baseFsHandle{root: h.Config.RootDir()}
	contents, err := handle.readFile(AuthConfigFile, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(contents), &cfg); err != nil {
		return nil, fmt.Errorf("unable to unmarshall auth config: %w", err)
	}
	return &cfg, nil
}
