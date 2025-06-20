// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package gateway

import (
	"github.com/foundriesio/dg-satellite/server"
	"github.com/labstack/echo/v4"
)

func authDevice(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		ctx := req.Context()
		log := server.CtxGetLog(ctx)
		tls := c.Request().TLS
		cert := tls.PeerCertificates[0]
		uuid := cert.Subject.CommonName

		// TODO look up device from DB, etc

		log = log.With("device", uuid)
		ctx = server.CtxWithLog(ctx, log)
		c.SetRequest(req.WithContext(ctx))

		return next(c)
	}
}
