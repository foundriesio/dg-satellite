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

	r, err := http.Get(fmt.Sprintf("http://%s/doesnotexist", server.ApiAddress()))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)
	require.Nil(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))
}
