// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/foundriesio/dg-satellite/server"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

var errTooManyRequests = errors.New("rate-limit exceeded. Try again later")
var errTooManyBadAuthOps = errors.New("too many bad authentication operations. Try again later")

func (cfg authConfigLocal) NewRateLimiter() *localAuthRateLimiter {
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

	return &localAuthRateLimiter{
		attemptsPerSecond:     cfg.AttemptsPerSecond,
		attemptsBlockDuration: time.Duration(cfg.AttemptsBlockDurationSec) * time.Second,
		badAuthLimit:          cfg.BadAuthLimit,
		badAuthBlockDuration:  time.Duration(cfg.BadAuthBlockDurationSec) * time.Second,
		badAuths:              make(map[string]*authLimiter),
		rateLimits:            make(map[string]*rateLimiter),
	}
}

// This implementation is similar to the echo rate limiter but done in a way that
// allows us to block IPs that have too many bad auth operations for a given amount of time
type localAuthRateLimiter struct {
	mutex sync.Mutex

	badAuths   map[string]*authLimiter
	rateLimits map[string]*rateLimiter

	lastGc time.Time

	attemptsPerSecond     int
	attemptsBlockDuration time.Duration
	badAuthLimit          int
	badAuthBlockDuration  time.Duration
}

type rateLimiter struct {
	*rate.Limiter
	lastSeen time.Time
}

type authLimiter struct {
	*rateLimiter
	blockUntil time.Time
}

func (rl *localAuthRateLimiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			identifier := c.RealIP()
			if err := rl.allow(identifier); err != nil {
				slog.Warn("Blocking IP", "ip", identifier, "err", err)
				return server.EchoError(c, err, http.StatusTooManyRequests, err.Error())
			}
			return next(c)
		}
	}
}

func (rl *localAuthRateLimiter) FlagBadOperation(c echo.Context) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	identifier := c.RealIP()
	v, exists := rl.badAuths[identifier]
	if !exists {
		// Allow badAuthLimit bad auth operations per minute before blocking,
		// rate-limiter does requests/second, this converts to requests/minute
		r := float64(1) / 60
		v = &authLimiter{
			rateLimiter: &rateLimiter{
				Limiter: rate.NewLimiter(rate.Limit(r), rl.badAuthLimit),
			},
		}
		rl.badAuths[identifier] = v
	}
	v.lastSeen = time.Now()
	allow := v.AllowN(v.lastSeen, 1)
	if !allow {
		v.blockUntil = time.Now().Add(rl.badAuthBlockDuration)
	}
}

func (rl *localAuthRateLimiter) allow(identifier string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	if time.Since(rl.lastGc) > 5*time.Minute {
		for id, v := range rl.badAuths {
			if time.Since(v.lastSeen) > 2*time.Minute && now.After(v.blockUntil) {
				delete(rl.badAuths, id)
			}
		}
		for id, v := range rl.rateLimits {
			if time.Since(v.lastSeen) > 2*time.Minute {
				delete(rl.rateLimits, id)
			}
		}
		rl.lastGc = now
	}

	// Check if this IP is already blocked due to bad auth operations
	v, exists := rl.badAuths[identifier]
	if exists && (now.Before(v.blockUntil) || v.Tokens() <= 0) {
		return errTooManyBadAuthOps
	}

	// Check the per-second rate limit; if exceeded, block the IP
	rv, rvExists := rl.rateLimits[identifier]
	if !rvExists {
		rv = &rateLimiter{
			Limiter: rate.NewLimiter(rate.Limit(rl.attemptsPerSecond), rl.attemptsPerSecond),
		}
		rl.rateLimits[identifier] = rv
	}
	rv.lastSeen = now
	if !rv.Allow() {
		if !exists {
			r := float64(1) / 60
			v = &authLimiter{
				rateLimiter: &rateLimiter{
					Limiter: rate.NewLimiter(rate.Limit(r), rl.badAuthLimit),
				},
			}
			rl.badAuths[identifier] = v
		}
		v.blockUntil = now.Add(rl.attemptsBlockDuration)
		return errTooManyRequests
	}

	return nil
}
