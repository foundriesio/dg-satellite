// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServe(t *testing.T) {
	tmpDir := t.TempDir()
	common := CommonArgs{
		DataDir: filepath.Join(tmpDir, "data"),
	}
	server := ServeCmd{}

	csr := CsrCmd{
		DnsName: "example.com",
		Factory: "example",
	}

	err := csr.Run(common)
	require.Nil(t, err)
	caKeyFile, caFile := createSelfSignedRoot(t, common)
	sign := CsrSignCmd{
		CaKey:  caKeyFile,
		CaCert: caFile,
		Csr:    filepath.Join(common.CertsDir(), "tls.csr"),
	}
	require.Nil(t, sign.Run(common))
	// create an empty ca file to make the server happy. no client will be able to handshake with it
	require.Nil(t, os.WriteFile(filepath.Join(common.CertsDir(), "cas.pem"), []byte{}, 0o744))

	go func() {
		require.Nil(t, server.Run(common))
	}()
	time.Sleep(time.Millisecond * 300)

	addr := server.apiServer.Listener.Addr().String()
	r, err := http.Get(fmt.Sprintf("http://%s/doesnotexist", addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)
	require.True(t, len(r.Header.Get("X-Request-Id")) > 0)

	addr = server.gatewayServer.TLSListener.Addr().String()
	_, err = http.Get(fmt.Sprintf("https://%s/doesnotexist", addr))
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "failed to verify certificate")

	server.quit <- syscall.SIGTERM
}
