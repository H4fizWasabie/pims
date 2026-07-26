package handler

import (
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

var loginLimiter = &rateLimiter{
	failures: make(map[string][]time.Time),
}

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
	return len(recent) < 5
}

func (rl *rateLimiter) record(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.failures[ip] = append(rl.failures[ip], time.Now())
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Strip port from RemoteAddr
	ip := r.RemoteAddr
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}
