// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package web

import (
	"github.com/foundriesio/dg-satellite/auth/providers"
	"github.com/foundriesio/dg-satellite/context"
)

type ctxKey int

const ctxKeySession ctxKey = iota

func CtxGetSession(ctx context.Context) *providers.Session {
	return ctx.Value(ctxKeySession).(*providers.Session)
}

func CtxWithSession(ctx context.Context, session *providers.Session) context.Context {
	return context.WithValue(ctx, ctxKeySession, session)
}
