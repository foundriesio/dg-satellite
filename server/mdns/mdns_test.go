// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package mdns

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
)

const (
	testPort uint16 = 9042
	testTld  string = "domain-that-makes.no-sense."
)

type testClient struct {
	srv  server.Server
	mdns server.Server
}

func newTestClient(t *testing.T) *testClient {
	log, err := context.InitLogger("debug")
	require.Nil(t, err)
	ctx := context.CtxWithLog(context.Background(), log)
	// Use Interface, not Address, so that the auto-detection is also verified
	mdns, err := NewServer(ctx, ServerParams{Intf: "lo", Tld: testTld, Port: testPort})
	require.Nil(t, err)
	srv := server.NewServer(ctx, server.NewEchoServer(), "test-server", testPort, nil)
	quitErr := make(chan error, 2)
	mdns.Start(quitErr)
	srv.Start(quitErr)
	time.Sleep(2 * time.Millisecond)
	_ = srv.GetAddress() // Waits until the server starts or fails to start.
	select {
	case err = <-quitErr:
		require.NotNil(t, err)
	default:
	}
	return &testClient{srv: srv, mdns: mdns}
}

func (c *testClient) Close() {
	c.mdns.Shutdown(2 * time.Millisecond)
	c.srv.Shutdown(2 * time.Millisecond)
}

func (c *testClient) GET(uri string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil); err != nil {
		return nil, err
	} else {
		client := http.Client{}
		return client.Do(req)
	}
}

func testDomain(subDomain string) string {
	return subDomain + "." + testTld
}

func testUri(subDomain string) string {
	return fmt.Sprintf("http://%s.%s:%d/does-not-matter", subDomain, testTld, testPort)
}

func assertLookup(t *testing.T, domain string) {
	// Verify that the domain name resolves to an IP address which reverse resolves to the given domain name.
	addrs, err := net.LookupHost(domain)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(addrs), addrs)
	if len(addrs) == 1 {
		// Only makes sense if the above assertion passed
		hosts, err := net.LookupAddr(addrs[0])
		assert.Nil(t, err)
		assert.Contains(t, hosts, domain)
	}
}

func TestRouting(t *testing.T) {
	// This test must be run inside a properly pre-configured container.
	// Otherwise, it would try to access the internet, and apparently fail.
	// See the `contrib/test-XXXdns.sh` for the test runners.
	if val := os.Getenv("TEST_READY"); val != "1" {
		return
	}

	c := newTestClient(t)

	// Test DNS all lookups point to the real hostname.
	_, err := net.LookupHost(testDomain("not-configured"))
	assert.NotNil(t, err)
	assertLookup(t, testDomain("api"))
	assertLookup(t, testDomain("hub"))
	assertLookup(t, testDomain("ostree"))

	// Test a real HTTP request is routed properly (double-checks the above name lookups in a different way).
	// Logic is extremely simple: if domain is not configured - we get an error; otherwise, no error.
	_, err = c.GET(testUri("not-configured"))
	assert.NotNil(t, err)
	_, err = c.GET(testUri("api"))
	assert.Nil(t, err)
	_, err = c.GET(testUri("hub"))
	assert.Nil(t, err)
	_, err = c.GET(testUri("ostree"))
	assert.Nil(t, err)
	c.Close()
}
