package storage

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	Limiter  *rate.Limiter
	LastSeen *time.Time
}

type RateLimiter struct {
	Visitors map[string]visitor
	Mu       sync.RWMutex
}

/* TODO: Implement rate limiter
	This is not required for the proof-of-context, and I realized that after starting it.
	The following code makes no sense:

func getLimiter(addr string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	limiter, ok := visitors[addr]
	if !ok {
		limiter = rate.NewLimiter(1, 3)
		visitors[addr] = limiter
	}
	return limiter
}
*/
