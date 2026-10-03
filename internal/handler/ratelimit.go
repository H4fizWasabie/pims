package handler

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	max      int
	failures map[string][]time.Time
}

// loginLimiter counts failed logins; demoLimiter counts every demo session start.
var (
	loginLimiter = &rateLimiter{max: 5, failures: make(map[string][]time.Time)}
	demoLimiter  = &rateLimiter{max: 20, failures: make(map[string][]time.Time)}
)

func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			loginLimiter.cleanup()
		}
	}()
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-1 * time.Minute)
	for ip, times := range rl.failures {
		var kept []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(rl.failures, ip)
		} else {
			rl.failures[ip] = kept
		}
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	times := rl.failures[ip]
	cutoff := time.Now().Add(-1 * time.Minute)
	var recent []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	rl.failures[ip] = recent
	return len(recent) < rl.max
}

func (rl *rateLimiter) record(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.failures[ip] = append(rl.failures[ip], time.Now())
}

func getClientIP(r *http.Request) string {
	// Only the LAST entry of X-Forwarded-For is trusted. Caddy appends the
	// real client IP to any client-supplied chain, so earlier entries are
	// attacker-controlled; trusting the first (or the whole chain) would
	// let an attacker rotate IPs and bypass the login rate limit.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.LastIndex(xff, ","); i >= 0 {
			return strings.TrimSpace(xff[i+1:])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
