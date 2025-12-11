// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/foundriesio/dg-satellite/auth"
	"github.com/foundriesio/dg-satellite/context"
)

type ctxKey int

const ctxKeySession ctxKey = iota

func CtxGetSession(ctx context.Context) *auth.Session {
	return ctx.Value(ctxKeySession).(*auth.Session)
}

func CtxWithSession(ctx context.Context, session *auth.Session) context.Context {
	return context.WithValue(ctx, ctxKeySession, session)
}

func CtxGetJson(ctx context.Context, resource string, result any) error {
	s := CtxGetSession(ctx)

	req, err := http.NewRequest("GET", s.BaseUrl+resource, nil)
	if err != nil {
		return err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			context.CtxGetLog(ctx).Error("unable to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: HTTP_%d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
