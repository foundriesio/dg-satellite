// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/foundriesio/dg-satellite/server"
	"github.com/foundriesio/dg-satellite/storage"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

var errTooManyRequests = errors.New("rate-limit exceeded")
var errTooManyBadAuthOps = errors.New("too many bad authentication operations")

func NewRateLimiter(cfg storage.RateLimitConfig) *authRateLimiter {
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

	badAuthSweepAge := 2 * time.Duration(cfg.BadAuthBlockDurationSec) * time.Second
	rateLimitSweepAge := 2 * time.Duration(cfg.AttemptsBlockDurationSec) * time.Second

	rl := &authRateLimiter{
		attemptsPerSecond:     cfg.AttemptsPerSecond,
		attemptsBlockDuration: time.Duration(cfg.AttemptsBlockDurationSec) * time.Second,
		badAuthLimit:          cfg.BadAuthLimit,
		badAuthBlockDuration:  time.Duration(cfg.BadAuthBlockDurationSec) * time.Second,
		badAuths:              newGenerationMap[*authLimiter](badAuthSweepAge),
		rateLimits:            newGenerationMap[*rateLimiter](rateLimitSweepAge),
	}
	rl.Middleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			identifier := c.RealIP()
			if err := rl.allow(identifier); err != nil {
				return server.EchoError(c, err, http.StatusTooManyRequests, err.Error())
			}
			return next(c)
		}
	}

	return rl
}

// This implementation is similar to the echo rate limiter but done in a way that
// allows us to block IPs that have too many bad auth operations for a given amount of time
type authRateLimiter struct {
	Middleware echo.MiddlewareFunc

	mutex sync.Mutex

	badAuths   generationMap[*authLimiter]
	rateLimits generationMap[*rateLimiter]

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
	blockUntil  time.Time
	blockReason error
}

func (rl *authRateLimiter) FlagBadOperation(c echo.Context) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	identifier := c.RealIP()
	v, exists := rl.badAuths.get(identifier)
	if !exists {
		// Allow badAuthLimit bad auth operations per minute before blocking,
		// rate-limiter does requests/second, this converts to requests/minute
		r := float64(1) / 60
		v = &authLimiter{
			rateLimiter: &rateLimiter{
				Limiter: rate.NewLimiter(rate.Limit(r), rl.badAuthLimit),
			},
		}
		rl.badAuths.put(identifier, v)
	}
	v.lastSeen = time.Now()
	allow := v.AllowN(v.lastSeen, 1)
	if !allow {
		v.blockUntil = time.Now().Add(rl.badAuthBlockDuration)
		isoTime := v.blockUntil.UTC().Format(time.RFC3339)
		v.blockReason = fmt.Errorf("%w: You are blocked until %s", errTooManyBadAuthOps, isoTime)
	}
}

func (rl *authRateLimiter) allow(identifier string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	rl.badAuths.sweep(now)
	rl.rateLimits.sweep(now)

	// Check if this IP is already blocked due to bad auth operations
	v, exists := rl.badAuths.get(identifier)
	if exists && (now.Before(v.blockUntil) || v.Tokens() <= 0) {
		return v.blockReason
	}

	// Check the per-second rate limit; if exceeded, block the IP
	rv, rvExists := rl.rateLimits.get(identifier)
	if !rvExists {
		rv = &rateLimiter{
			Limiter: rate.NewLimiter(rate.Limit(rl.attemptsPerSecond), rl.attemptsPerSecond),
		}
		rl.rateLimits.put(identifier, rv)
	}
	rv.lastSeen = now
	if !rv.Allow() {
		if !exists {
			r := float64(1) / 60 // Convert attempts per second to attempts per minute for bad auth limiter
			v = &authLimiter{
				rateLimiter: &rateLimiter{
					Limiter: rate.NewLimiter(rate.Limit(r), rl.badAuthLimit),
				},
			}
			rl.badAuths.put(identifier, v)
		}
		v.blockUntil = now.Add(rl.attemptsBlockDuration)
		isoTime := v.blockUntil.UTC().Format(time.RFC3339)
		v.blockReason = fmt.Errorf("%w: You are blocked until %s", errTooManyRequests, isoTime)
		return v.blockReason
	}

	return nil
}
