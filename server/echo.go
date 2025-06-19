// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// All rights reserved.
// Confidential and Proprietary - Qualcomm Technologies, Inc.
package server

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/random"
)

func NewEchoServer(name string, logger *slog.Logger) *echo.Echo {
	server := echo.New()
	server.HideBanner = true
	server.Use(contextLogger(logger))
	server.Use(middlewareLogger(name))

	return server
}

func middlewareLogger(name string) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		HandleError:      true, // forwards error to the global error handler, so it can decide appropriate status code
		LogContentLength: true,
		LogError:         true,
		LogMethod:        true,
		LogStatus:        true,
		LogURI:           true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log := CtxGetLog(c.Request().Context())
			if v.Error == nil {
				log.LogAttrs(context.Background(), slog.LevelInfo, name,
					slog.String("method", v.Method),
					slog.String("content-length", v.ContentLength),
					slog.Int("status", v.Status),
				)
			} else {
				log.LogAttrs(context.Background(), slog.LevelError, name,
					slog.String("method", v.Method),
					slog.String("content-length", v.ContentLength),
					slog.Int("status", v.Status),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	})
}

func contextLogger(log *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			ctx := req.Context()

			rid := req.Header.Get(echo.HeaderXRequestID)
			if rid == "" {
				rid = random.String(12) // No need for uuid, save some space
			}
			res.Header().Set(echo.HeaderXRequestID, rid)
			log = log.With("req_id", rid, "uri", req.RequestURI)
			ctx = CtxWithLog(ctx, log)
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	}
}
