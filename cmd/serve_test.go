// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package main

import (
	"fmt"
	"net/http"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/foundriesio/dg-satellite/context"
)

func TestServe(t *testing.T) {
	common := CommonArgs{}
	server := NewServeCmd()

	log, err := context.InitLogger("debug")
	require.Nil(t, err)
	ctx := context.CtxWithLog(context.Background(), log)

	go func() {
		require.Nil(t, server.Run(ctx, common))
	}()
	server.WaitUntilStarted()

	r, err := http.Get(fmt.Sprintf("http://%s/doesnotexist", server.ApiAddress))
	require.Nil(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)
	require.Nil(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))
}
