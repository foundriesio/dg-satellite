// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package context

import (
	"context"
	"log/slog"
)

type (
	Context = context.Context
	ctxKey  int
)

var (
	Background  = context.Background
	WithTimeout = context.WithTimeout
)

const (
	ctxKeyLogger ctxKey = iota
)

func CtxGetLog(ctx context.Context) *slog.Logger {
	return ctx.Value(ctxKeyLogger).(*slog.Logger)
}

func CtxWithLog(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger, log)
}
