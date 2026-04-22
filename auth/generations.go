// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package auth

import "time"

// generationMap is a two-generation map that provides O(1) garbage collection.
// Active entries are promoted to `front` on access. Periodically, `front` is
// rotated to `back` and a fresh `front` is allocated, discarding the old `back`.
type generationMap[V any] struct {
	front    map[string]V
	back     map[string]V
	lastSwap time.Time
	sweepAge time.Duration
}

func newGenerationMap[V any](sweepAge time.Duration) generationMap[V] {
	return generationMap[V]{
		front:    make(map[string]V),
		back:     make(map[string]V),
		lastSwap: time.Now(),
		sweepAge: sweepAge,
	}
}

// get looks up a key in front, then back. If found in back, it is promoted to front.
func (g *generationMap[V]) get(key string) (V, bool) {
	if v, ok := g.front[key]; ok {
		return v, true
	}
	if v, ok := g.back[key]; ok {
		g.front[key] = v
		delete(g.back, key)
		return v, true
	}
	var zero V
	return zero, false
}

// put inserts or updates a key in front.
func (g *generationMap[V]) put(key string, v V) {
	g.front[key] = v
}

// sweep rotates generations if sweepAge has elapsed since the last swap.
func (g *generationMap[V]) sweep(now time.Time) {
	if now.Sub(g.lastSwap) >= g.sweepAge {
		g.back = g.front
		g.front = make(map[string]V, (len(g.back)+len(g.front))/2)
		g.lastSwap = now
	}
}
