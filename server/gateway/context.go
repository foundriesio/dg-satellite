// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"github.com/foundriesio/dg-satellite/context"
	storage "github.com/foundriesio/dg-satellite/storage/gateway"
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
	ctxKeyDevice ctxKey = iota
)

func CtxGetDevice(ctx context.Context) *storage.Device {
	return ctx.Value(ctxKeyDevice).(*storage.Device)
}

func CtxWithDevice(ctx Context, device *storage.Device) Context {
	return context.WithValue(ctx, ctxKeyDevice, device)
}
