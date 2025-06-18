// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCsr(t *testing.T) {
	tmpDir := t.TempDir()

	csr := CsrCmd{
		DnsName: "example.com",
		Factory: "example",
	}

	// fail because we require a new directory (so we don't accidentally overwrite our key)
	common := CommonArgs{
		DataDir: tmpDir,
	}
	err := csr.Run(common)
	require.NotNil(t, err)
	require.True(t, errors.Is(err, os.ErrExist))

	common.DataDir = filepath.Join(common.DataDir, "data")
	require.Nil(t, csr.Run(common))

	// Create a root CA
	caKeyFile := filepath.Join(common.CertsDir(), "tls.key") // just steal the key we already generated
	key, err := loadKey(caKeyFile)
	require.Nil(t, err)

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"example"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDer, err := x509.CreateCertificate(rand.Reader, ca, ca, &key.PublicKey, key)
	require.Nil(t, err)
	caPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caDer,
		},
	)
	caFile := filepath.Join(tmpDir, "ca.crt")
	require.Nil(t, os.WriteFile(caFile, caPem, 0o744))

	sign := CsrSignCmd{
		CaKey:  caKeyFile,
		CaCert: caFile,
		Csr:    filepath.Join(common.CertsDir(), "tls.csr"),
	}
	require.Nil(t, sign.Run(common))

	cert, err := loadCert(filepath.Join(common.CertsDir(), "tls.crt"))
	require.Nil(t, err)
	require.Equal(t, "example.com", cert.Subject.CommonName)
}
