// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/foundriesio/dg-satellite/context"
	"github.com/foundriesio/dg-satellite/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

var errTooManyBadAuthOps = errors.New("too many bad authentication operations")

func (cfg authConfigLocal) NewRateLimiter() (*localAuthRateLimiter, echo.MiddlewareFunc) {
	attemptsPerSecond := cfg.AttemptsPerSecond
	if attemptsPerSecond <= 0 {
		attemptsPerSecond = 2
	}

	rl := &localAuthRateLimiter{
		visitors: make(map[string]*visitor),
		store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(attemptsPerSecond),
				ExpiresIn: 2 * time.Minute,
			},
		),
	}

	rlConfig := middleware.RateLimiterConfig{
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			time.Sleep(2 * time.Second) // slow down responses to make brute-force attacks less effective
			if errors.Is(err, errTooManyBadAuthOps) {
				return server.EchoError(context, err, http.StatusTooManyRequests, "Too many bad authentication attempts. Please try again later.")
			}
			return middleware.DefaultRateLimiterConfig.DenyHandler(context, identifier, err)
		},
		Store: rl,
	}
	return rl, middleware.RateLimiterWithConfig(rlConfig)
}

// This implementation is similar to the echo rate limiter but done in a way that
// allows us to block IPs that have too many bad auth operations for a given amount of time
type localAuthRateLimiter struct {
	visitors map[string]*visitor
	mutex    sync.Mutex

	lastGc time.Time
	store  middleware.RateLimiterStore
}

type visitor struct {
	*rate.Limiter
	lastSeen   time.Time
	blockUntil time.Time
}

func (rl *localAuthRateLimiter) FlagBadOperation(c echo.Context) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	identifier := c.RealIP()
	v, exists := rl.visitors[identifier]
	if !exists {
		// Allow 5 bad auth operations per minute before blocking,
		// rate-limiter doesn requests/second, this converts to requests/minute
		r := float64(1) / 60
		v = &visitor{
			Limiter:  rate.NewLimiter(rate.Limit(r), 5),
			lastSeen: time.Now(),
		}
		rl.visitors[identifier] = v
	}
	v.lastSeen = time.Now()
	allow := v.AllowN(v.lastSeen, 1)
	if !allow {
		context.CtxGetLog(c.Request().Context()).Warn("Too many bad auth operations. Blocking IP", "ip", identifier)
		v.blockUntil = time.Now().Add(5 * time.Minute)
	}
}

func (rl *localAuthRateLimiter) Allow(identifier string) (bool, error) {
	if allow, err := rl.store.Allow(identifier); !allow {
		return false, err
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	if time.Since(rl.lastGc) > 5*time.Minute {
		for id, v := range rl.visitors {
			// we flag operations by the minute, so 2 minutes is plenty
			if time.Since(v.lastSeen) > 2*time.Minute && now.After(v.blockUntil) {
				delete(rl.visitors, id)
			}
		}
		rl.lastGc = now
	}

	visitor, exists := rl.visitors[identifier]
	if !exists {
		return true, nil
	}
	if now.Before(visitor.blockUntil) {
		return false, fmt.Errorf("%w. IP: %s", errTooManyBadAuthOps, identifier)
	}
	if visitor.Tokens() > 0 {
		return true, nil
	}
	return false, fmt.Errorf("%w. IP: %s", errTooManyBadAuthOps, identifier)
}
