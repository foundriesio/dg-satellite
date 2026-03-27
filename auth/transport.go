// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"log/slog"
	"net/http"
)

func newHttpClientWithInternalToken(token string) *http.Client {
	return &http.Client{
		Transport: &roundTripper{
			base:  http.DefaultTransport,
			token: token,
		},
	}
}

type roundTripper struct {
	base  http.RoundTripper
	token string
}

func (t roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
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
	req2.Header.Set("Authorization", "X-Internal "+t.token)

	// req.Body is assumed to be closed by the base RoundTripper.
	reqBodyClosed = true
	return t.base.RoundTrip(req2)
}
