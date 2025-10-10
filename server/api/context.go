// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package api

import (
	"github.com/foundriesio/dg-satellite/context"
)

type (
	Context = context.Context
	ctxKey  int
)

var (
	CtxGetLog  = context.CtxGetLog
	CtxWithLog = context.CtxWithLog
)

const (
	ctxKeyProd ctxKey = iota
)

func CtxGetIsProd(ctx Context) bool {
	return ctx.Value(ctxKeyProd).(bool)
}

func CtxWithIsProd(ctx Context, isProd bool) Context {
	return context.WithValue(ctx, ctxKeyProd, isProd)
}
