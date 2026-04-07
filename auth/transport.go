// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"log/slog"
	"net/http"
)

func newHttpClientWithSessionCookie(cookie *http.Cookie) *http.Client {
	return &http.Client{
		Transport: &cookieRoundTripper{
			base:   http.DefaultTransport,
			cookie: cookie,
		},
	}
}

type cookieRoundTripper struct {
	base   http.RoundTripper
	cookie *http.Cookie
}

func (t cookieRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBodyClosed := false
	if req.Body != nil {
		defer func() {
			if !reqBodyClosed {
				if err := req.Body.Close(); err != nil {
					slog.Error("failed to close request body", "error", err)
				}
			}
		}()
	}

	req2 := req.Clone(req.Context())
	req2.AddCookie(t.cookie)

	// req.Body is assumed to be closed by the base RoundTripper.
	reqBodyClosed = true
	return t.base.RoundTrip(req2)
}
