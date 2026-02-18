// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func (cfg authConfigLocal) NewRateLimiter() echo.MiddlewareFunc {
	attemptsPerSecond := cfg.AttemptsPerSecond
	if attemptsPerSecond <= 0 {
		attemptsPerSecond = 2
	}

	rlConfig := middleware.RateLimiterConfig{
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			time.Sleep(2 * time.Second) // slow down responses to make brute-force attacks less effective
			return middleware.DefaultRateLimiterConfig.DenyHandler(context, identifier, err)
		},
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(attemptsPerSecond),
				ExpiresIn: 2 * time.Minute,
			},
		),
	}
	return middleware.RateLimiterWithConfig(rlConfig)
}
