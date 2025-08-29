// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/random"

	"github.com/foundriesio/dg-satellite/context"
)

func NewEchoServer() *echo.Echo {
	server := echo.New()
	server.HideBanner = true
	server.HidePort = true
	server.Use(contextLogger())
	server.Use(middlewareLogger())
	return server
}

func middlewareLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		HandleError:      true, // forwards error to the global error handler, so it can decide appropriate status code
		LogContentLength: true,
		LogError:         true,
		LogMethod:        true,
		LogStatus:        true,
		LogURI:           true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log := context.CtxGetLog(c.Request().Context())
			args := []any{"method", v.Method, "content-length", v.ContentLength, "status", v.Status}
			if v.Error == nil {
				log.Info("response", args...)
			} else {
				args = append(args, "err", v.Error.Error())
				log.Error("response", args...)
			}
			return nil
		},
	})
}

func contextLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			ctx := req.Context()
			log := context.CtxGetLog(ctx)

			rid := req.Header.Get(echo.HeaderXRequestID)
			if rid == "" {
				rid = random.String(12) // No need for uuid, save some space
			}
			res.Header().Set(echo.HeaderXRequestID, rid)
			log = log.With("req_id", rid, "uri", req.RequestURI)
			ctx = context.CtxWithLog(ctx, log)
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	}
}
