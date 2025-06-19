// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package server

import (
	"context"
	"log/slog"
)

type (
	ctxKey int
)

const (
	CtxKeyLogger = ctxKey(3)
)

func CtxGetLog(ctx context.Context) *slog.Logger {
	return ctx.Value(CtxKeyLogger).(*slog.Logger)
}

func CtxWithLog(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, CtxKeyLogger, log)
}
