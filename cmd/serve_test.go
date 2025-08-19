// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/storage"
)

func TestServe(t *testing.T) {
	tmpDir := t.TempDir()
	common := CommonArgs{
		DataDir: filepath.Join(tmpDir, "data"),
	}
	fs, err := storage.NewFs(common.DataDir)
	require.Nil(t, err)
	apiAddress := ""
	gatewayAddress := ""
	startedWg := sync.WaitGroup{}
	startedWg.Add(1)
	server := ServeCmd{
		startedCb: func(apiAddr, gwAddr string) {
			apiAddress = apiAddr
			gatewayAddress = gwAddr
			startedWg.Done()
		},
	}

	log, err := context.InitLogger("debug")
	require.Nil(t, err)
	common.ctx = context.CtxWithLog(context.Background(), log)

	csr := CsrCmd{
		DnsName: "example.com",
		Factory: "example",
	}

	err = csr.Run(common)
	require.Nil(t, err)
	caKeyFile, caFile := createSelfSignedRoot(t, fs)
	sign := CsrSignCmd{
		CaKey:  caKeyFile,
		CaCert: caFile,
		Csr:    filepath.Join(fs.Config.CertsDir(), "tls.csr"),
	}
	require.Nil(t, sign.Run(common))
	// create an empty ca file to make the server happy. no client will be able to handshake with it
	require.Nil(t, os.WriteFile(filepath.Join(fs.Config.CertsDir(), "cas.pem"), []byte{}, 0o744))

	go func() {
		require.Nil(t, server.Run(common))
	}()
	startedWg.Wait()

	r, err := http.Get(fmt.Sprintf("http://%s/doesnotexist", apiAddress))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)
	require.Equal(t, 12, len(r.Header.Get("X-Request-Id")))

	_, err = http.Get(fmt.Sprintf("https://%s/doesnotexist", gatewayAddress))
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "failed to verify certificate")

	require.Nil(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))
}
