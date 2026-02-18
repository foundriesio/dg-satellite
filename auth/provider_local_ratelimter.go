// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"errors"
	"log/slog"
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
	if cfg.AttemptsPerSecond <= 0 {
		cfg.AttemptsPerSecond = 2
	}
	if cfg.AttemptsBlockDurationSec <= 0 {
		cfg.AttemptsBlockDurationSec = 30
	}
	if cfg.BadAuthLimit <= 0 {
		cfg.BadAuthLimit = 5
	}
	if cfg.BadAuthBlockDurationSec <= 0 {
		cfg.BadAuthBlockDurationSec = 300
	}

	rl := &localAuthRateLimiter{
		attemptsBlockDuration: time.Duration(cfg.AttemptsBlockDurationSec) * time.Second,
		badAuthLimit:          cfg.BadAuthLimit,
		badAuthBlockDuration:  time.Duration(cfg.BadAuthBlockDurationSec) * time.Second,
		badAuths:              make(map[string]*visitor),
		store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(cfg.AttemptsPerSecond),
				ExpiresIn: 2 * time.Minute,
			},
		),
	}

	rlConfig := middleware.RateLimiterConfig{
		DenyHandler: func(context echo.Context, identifier string, err error) error {
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
	badAuths      map[string]*visitor
	badAuthsMutex sync.Mutex

	lastGc time.Time
	store  middleware.RateLimiterStore

	attemptsBlockDuration time.Duration
	badAuthLimit          int
	badAuthBlockDuration  time.Duration
}

type visitor struct {
	*rate.Limiter
	lastSeen   time.Time
	blockUntil time.Time
}

func (rl *localAuthRateLimiter) FlagBadOperation(c echo.Context) {
	rl.badAuthsMutex.Lock()
	defer rl.badAuthsMutex.Unlock()

	identifier := c.RealIP()
	v, exists := rl.badAuths[identifier]
	if !exists {
		// Allow 5 bad auth operations per minute before blocking,
		// rate-limiter doesn requests/second, this converts to requests/minute
		r := float64(1) / 60
		v = &visitor{
			Limiter: rate.NewLimiter(rate.Limit(r), rl.badAuthLimit),
		}
		rl.badAuths[identifier] = v
	}
	v.lastSeen = time.Now()
	allow := v.AllowN(v.lastSeen, 1)
	if !allow {
		v.blockUntil = time.Now().Add(rl.badAuthBlockDuration)
		context.CtxGetLog(c.Request().Context()).Warn("Too many bad auth operations. Blocking IP", "ip", identifier, "until", v.blockUntil)
	}
}

func (rl *localAuthRateLimiter) Allow(identifier string) (bool, error) {
	rl.badAuthsMutex.Lock()
	defer rl.badAuthsMutex.Unlock()

	now := time.Now()
	if time.Since(rl.lastGc) > 5*time.Minute {
		for id, v := range rl.badAuths {
			// we flag operations by the minute, so 2 minutes is plenty
			if time.Since(v.lastSeen) > 2*time.Minute && now.After(v.blockUntil) {
				delete(rl.badAuths, id)
			}
		}
		rl.lastGc = now
	}

	// Check if this IP is already blocked
	v, exists := rl.badAuths[identifier]
	if exists && (now.Before(v.blockUntil) || v.Tokens() <= 0) {
		return false, errTooManyBadAuthOps
	}

	// Check the per-second rate limit; if exceeded, block the IP for 1 minute
	if allow, err := rl.store.Allow(identifier); !allow {
		if !exists {
			r := float64(1) / 60
			v = &visitor{
				Limiter:  rate.NewLimiter(rate.Limit(r), rl.badAuthLimit),
				lastSeen: now,
			}
			rl.badAuths[identifier] = v
		}
		v.blockUntil = now.Add(rl.attemptsBlockDuration)
		slog.Warn("Rate limit exceeded. Blocking IP", "ip", identifier, "until", v.blockUntil, "error", err)
		return false, errTooManyBadAuthOps
	}

	return true, nil
}
