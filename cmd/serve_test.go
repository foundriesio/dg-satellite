// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"fmt"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServe(t *testing.T) {
	common := CommonArgs{}
	server := ServeCmd{}

	go func() {
		require.Nil(t, server.Run(common))
	}()
	time.Sleep(time.Millisecond * 300)

	addr := server.apiServer.Listener.Addr().String()
	r, err := http.Get(fmt.Sprintf("http://%s/doesnotexist", addr))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)
	require.True(t, len(r.Header.Get("X-Request-Id")) > 0)
	server.quit <- syscall.SIGTERM
}
