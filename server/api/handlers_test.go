// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package api

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/stretchr/testify/require"
)

type client struct {
	srv *httptest.Server
}

func (c client) GET(t *testing.T, resource string, status int) []byte {
	url := c.srv.URL + resource
	res, err := c.srv.Client().Get(url)
	require.Nil(t, err)
	buf, err := io.ReadAll(res.Body)
	require.Nil(t, err)
	require.Equal(t, status, res.StatusCode, string(buf))
	return buf
}

func testWrapper(t *testing.T, testFunc func(client)) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	e := server.NewEchoServer("api-test", logger)

	srv := httptest.NewUnstartedServer(e)
	RegisterHandlers(e, auth.FakeAuthUser)

	srv.StartTLS()
	t.Cleanup(srv.Close)

	c := client{
		srv: srv,
	}
	testFunc(c)
}

func TestApi(t *testing.T) {
	testWrapper(t, func(tc client) {
		_ = tc.GET(t, "/tmp", 200)
		_ = tc.GET(t, "/tmp?deny-has-scope=1", 403)
	})
}
