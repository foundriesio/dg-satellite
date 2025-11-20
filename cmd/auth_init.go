// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package main

import (
	"crypto/rand"
	"fmt"

	"github.com/foundriesio/dg-satellite/storage"
)

type AuthInitCmd struct {
}

func (c AuthInitCmd) Run(args CommonArgs) error {
	fs, err := storage.NewFs(args.DataDir)
	if err != nil {
		return err
	}

	if _, err := fs.Certs.ReadFile("hmac.secret"); err != nil {
		fmt.Println("Initializing new HMAC secret")
		secret := make([]byte, 64)
		if _, err := rand.Read(secret); err != nil {
			return fmt.Errorf("generating HMAC secret: %w", err)
		}
		if err := fs.Certs.WriteFile("hmac.secret", secret); err != nil {
			return fmt.Errorf("storing HMAC secret: %w", err)
		}
	}
	return nil
}
