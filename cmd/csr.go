// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/foundriesio/dg-satellite/storage"
)

type CsrCmd struct {
	DnsName string `arg:"required" help:"DNS host name devices address this gateway with"`
	Factory string `arg:"required"`
}

func (c CsrCmd) Run(args CommonArgs) error {
	fs, err := storage.NewFs(args.DataDir)
	if err != nil {
		return err
	}
	if err = os.Mkdir(fs.Config.CertsDir(), 0o740); err != nil {
		return fmt.Errorf("unable to create certs directory: %w", err)
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("unexpected error generating private key for CSR: %w", err)
	}

	subj := pkix.Name{
		CommonName:         c.DnsName,
		OrganizationalUnit: []string{c.Factory},
	}

	template := x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		DNSNames:           []string{c.DnsName},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, priv)
	if err != nil {
		return fmt.Errorf("unexpected error creating CSR: %w", err)
	}

	keyFile := filepath.Join(fs.Config.CertsDir(), "tls.key")
	privDer, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("unexpected error encoding private key: %w", err)
	}
	privPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: privDer,
		},
	)
	if err := os.WriteFile(keyFile, privPem, 0o740); err != nil {
		return fmt.Errorf("unable to store TLS private key for CSR: %w", err)
	}

	csrFile := filepath.Join(fs.Config.CertsDir(), "tls.csr")
	fd, err := os.Create(csrFile)
	if err != nil {
		return fmt.Errorf("unable to write TLS CSR: %w", err)
	}
	defer func() {
		if err := fd.Close(); err != nil {
			fmt.Println("Unexpected error closing file", err)
		}
	}()
	if err := pem.Encode(fd, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes}); err != nil {
		return fmt.Errorf("unexpected error in pem-encode: %w", err)
	}
	fmt.Println("CSR written to:", csrFile)
	if err := pem.Encode(os.Stdout, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes}); err != nil {
		return fmt.Errorf("unexpected error in pem-encode: %w", err)
	}

	return nil
}
