package handler

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	RATEMAX   = 1
	RATEBURST = 3
)

type Visitor struct {
	Limiter  *rate.Limiter
	LastSeen time.Time
}

type RateLimiter struct {
	lastCleaned time.Time

	mu       sync.Mutex
	visitors map[string]*Visitor
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		visitors:    make(map[string]*Visitor),
		lastCleaned: time.Now(),
	}
}

func (rl *RateLimiter) getVisitor(addr string) *Visitor {
	visitor, ok := rl.visitors[addr]
	if !ok {
		limiter := rate.NewLimiter(RATEMAX, RATEBURST)
		seen := time.Now()
		visitor = &Visitor{
			Limiter:  limiter,
			LastSeen: seen,
		}
		rl.visitors[addr] = visitor
	}
	return visitor
}

func (rl *RateLimiter) clean(maxAge time.Duration, except string) {
	if time.Since(rl.lastCleaned) < maxAge {
		return
	}
	rl.lastCleaned = time.Now()
	for addr, visitor := range rl.visitors {
		if addr == except {
			continue
		}
		if time.Since(visitor.LastSeen) > maxAge {
			delete(rl.visitors, addr)
		}
	}
}

func (rl *RateLimiter) Allow(addr string) bool {
	rl.mu.Lock()
	rl.clean(5*time.Minute, addr)
	now := time.Now()
	visitor := rl.getVisitor(addr)
	visitor.LastSeen = now
	allow := visitor.Limiter.Allow()
	rl.mu.Unlock()
	return allow
}
