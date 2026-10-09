//go:build !js && !wasm

package main

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ============================================================
// RateLimiterService — Pillar 1-C: Rate Limiting & DDoS Mitigation
// Uses token bucket + sliding window over existing RateBucket struct.
// ============================================================

// RateLimitTier defines a category of endpoints with shared quotas.
type RateLimitTier struct {
	Name           string
	MaxTokens      float64       // Bucket capacity (burst allowance)
	RefillRate     float64       // Tokens added per second
	WindowDuration time.Duration // Sliding window for additional cap
	MaxRequests    int           // Max requests within the sliding window
}

// WalletQuota maps a wallet address to its custom rate limit tier.
type WalletQuota struct {
	Wallet   string
	TierName string
	IsAdmin  bool        // Task 3105: infinite quota for admin wallets
	IPFallback string    // Task 3104: IP used when wallet detection fails
}

// RateLimiterService manages rate limiting across all API endpoints.
type RateLimiterService struct {
	Mu                 sync.RWMutex
	Lobby              *Lobby            // Reference for telemetry and config
	Tiers              map[string]*RateLimitTier // Configured tiers
	WalletQuotas       map[string]*WalletQuota   // Per-wallet quota assignments (Task 3103)
	AdminWallets       map[string]bool           // Task 3105: admin bypass list
	ClientIPs          sync.Map                  // wallet -> IP mapping for fallback
	RateLimitedCounter prometheus.Counter        // Task 3106: Prometheus counter
}

// task3101 — Default tier configuration (token bucket + sliding window)
var defaultTiers = map[string]*RateLimitTier{
	"auth": {
		Name:           "auth",
		MaxTokens:      5.0,       // 5 requests burst
		RefillRate:     1.0 / 12.0, // 1 token every 12s → ~5/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    5,
	},
	"economy": {
		Name:           "economy",
		MaxTokens:      10.0, // 10 requests burst
		RefillRate:     1.0 / 6.0, // 1 token every 6s → ~10/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    10,
	},
	"economy-tight": {
		Name:           "economy-tight",
		MaxTokens:      3.0, // 3 requests burst (high-risk payouts)
		RefillRate:     1.0 / 20.0, // 1 token every 20s → ~3/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    3,
	},
	"core-economy": {
		Name:           "core-economy",
		MaxTokens:      5.0, // 5 requests burst (leaderboard, career)
		RefillRate:     1.0 / 12.0, // 1 token every 12s → ~5/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    5,
	},
	"standard": {
		Name:           "standard",
		MaxTokens:      10.0, // 10 requests burst (card-stats, auctions)
		RefillRate:     1.0 / 6.0, // 1 token every 6s → ~10/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    10,
	},
	"wallet-default": {
		Name:           "wallet-default",
		MaxTokens:      30.0, // 30 requests burst (status, re-sync, admin)
		RefillRate:     1.0, // 1 token per second → ~60/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    60,
	},
	"achievement": {
		Name:           "achievement",
		MaxTokens:      15.0, // 15 requests burst (trophy endpoints)
		RefillRate:     1.0 / 4.0, // 1 token every 4s → ~15/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    15,
	},
	"underworld": {
		Name:           "underworld",
		MaxTokens:      8.0, // 8 requests burst (contracts, missions)
		RefillRate:     1.0 / 7.5, // 1 token every 7.5s → ~8/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    8,
	},
	"admin": {
		Name:           "admin",
		MaxTokens:      2.0, // 2 requests burst
		RefillRate:     1.0 / 30.0, // 1 token every 30s → ~2/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    2,
	},
	"default": {
		Name:           "default",
		MaxTokens:      20.0, // 20 requests burst (catch-all)
		RefillRate:     1.0 / 3.0, // 1 token every 3s → ~20/min sustained
		WindowDuration: 1 * time.Minute,
		MaxRequests:    20,
	},
}

// rateLimitedCounter is the Prometheus counter for Task 3106
var rateLimitedCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "api_rate_limited_total",
	Help: "Total number of API requests denied due to rate limiting.",
})

// NewRateLimiterService creates and initializes the rate limiter.
func NewRateLimiterService(lobby *Lobby, adminWallets []string) *RateLimiterService {
	rls := &RateLimiterService{
		Lobby:        lobby,
		Tiers:        make(map[string]*RateLimitTier),
		WalletQuotas: make(map[string]*WalletQuota),
		AdminWallets: make(map[string]bool),
	}

	// Copy default tiers (Task 3101)
	for name, tier := range defaultTiers {
		rls.Tiers[name] = &RateLimitTier{
			Name:           tier.Name,
			MaxTokens:    tier.MaxTokens,
			RefillRate:   tier.RefillRate,
			WindowDuration: tier.WindowDuration,
			MaxRequests:  tier.MaxRequests,
		}
	}

	// Task 3105: admin bypass wallets
	for _, w := range adminWallets {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" {
			rls.AdminWallets[w] = true
		}
	}

	// Task 3106: initialize Prometheus counter in telemetry (reuse existing TelemetryLogger if available)
	rls.RateLimitedCounter = rateLimitedCounter

	return rls
}

// SetWalletQuota assigns a per-wallet rate limit tier (Task 3103).
func (rls *RateLimiterService) SetWalletQuota(wallet, tierName string) {
	rls.Mu.Lock()
	defer rls.Mu.Unlock()

	wallet = strings.ToLower(strings.TrimSpace(wallet))
	tier, exists := defaultTiers[tierName]
	if !exists {
		tier = defaultTiers["default"]
	}

	rls.WalletQuotas[wallet] = &WalletQuota{
		Wallet:   wallet,
		TierName: tier.Name,
		IsAdmin:  rls.AdminWallets[wallet],
	}
}

// SetClientIP records the IP for a wallet (used for Task 3104 fallback).
func (rls *RateLimiterService) SetClientIP(wallet, ip string) {
	rls.ClientIPs.Store(wallet, ip)
}

// getClientIP extracts the client IP from the request.
func (rls *RateLimiterService) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (behind proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP (nginx pattern)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// getWalletFromRequest extracts wallet address from request headers or query params.
func (rls *RateLimiterService) getWalletFromRequest(r *http.Request) string {
	// Try header first
	wallet := r.Header.Get("X-Wallet-Address")
	if wallet != "" {
		return strings.ToLower(strings.TrimSpace(wallet))
	}
	// Try query param
	wallet = r.URL.Query().Get("wallet")
	if wallet != "" {
		return strings.ToLower(strings.TrimSpace(wallet))
	}
	return ""
}

// resolveKey returns the rate limit key for a request.
// Priority: admin wallet → per-wallet quota → IP fallback → path-based default.
// Returns (key string, shouldSkipLimiting bool).
func (rls *RateLimiterService) resolveKey(r *http.Request) (string, bool) {
	wallet := rls.getWalletFromRequest(r)

	// Task 3105: Admin wallets get infinite quota (skip limiting)
	if wallet != "" && rls.AdminWallets[wallet] {
		return "admin:" + wallet, true // skipped by limiter
	}

	// Task 3103: Per-wallet limits (if explicitly configured)
	if wallet != "" {
		if q, exists := rls.WalletQuotas[wallet]; exists {
			return "wallet:" + wallet, !q.IsAdmin
		}
	}

	// Task 3104: IP-based fallback when wallet detection fails
	ip := rls.getClientIP(r)
	if ip != "" && wallet == "" {
		return "ip:" + ip, true
	}

	// Fallback to path-based default tier names
	path := strings.ToLower(r.URL.Path)
	if strings.HasPrefix(path, "/api/auth/") || strings.HasPrefix(path, "/api/wallet/link") {
		return "tier:auth", false
	}
	if strings.HasPrefix(path, "/api/admin/") {
		return "tier:admin", false
	}
	if strings.HasPrefix(path, "/api/") {
		return "tier:economy", false
	}

	return "tier:default", false
}

// resolveTier returns the rate limit tier for a given key.
func (rls *RateLimiterService) resolveTier(key string) *RateLimitTier {
	if strings.HasPrefix(key, "wallet:") {
		wallet := strings.TrimPrefix(key, "wallet:")
		wallet = strings.ToLower(wallet)
		if q, exists := rls.WalletQuotas[wallet]; exists && q.TierName != "" {
			if t, ok := rls.Tiers[q.TierName]; ok {
				return t
			}
		}
	}
	if strings.HasPrefix(key, "tier:") {
		tierName := strings.TrimPrefix(key, "tier:")
		if t, ok := rls.Tiers[tierName]; ok {
			return t
		}
	}
	if strings.HasPrefix(key, "ip:") || key == "" {
		return rls.Tiers["default"]
	}
	return rls.Tiers["default"]
}

// Allow checks if the request is within rate limits.
// Returns (allowed bool, retryAfterSeconds float64).
func (rls *RateLimiterService) Allow(r *http.Request) (bool, float64) {
	key, skip := rls.resolveKey(r)
	if skip {
		return true, 0
	}

	tier := rls.resolveTier(key)
	if tier == nil {
		return true, 0 // No tier configured → allow
	}

	now := time.Now()

	// Sliding window check (prevents burst overflow within window)
	rls.Mu.Lock()
	windowCount := 0
	for k, v := range rls.Lobby.httpRateLimits {
		if !strings.HasPrefix(k, tier.Name+"_") {
			continue
		}
		if now.Sub(v.LastUpdate) <= tier.WindowDuration {
			windowCount++
		} else {
			delete(rls.Lobby.httpRateLimits, k) // Clean expired entries
		}
	}

	if windowCount >= tier.MaxRequests {
		rls.Mu.Unlock()
		return false, tier.WindowDuration.Seconds()
	}

	// Token bucket check (handles burst)
	bucketKey := key + "_bucket"
	bucket, exists := rls.Lobby.httpRateLimits[bucketKey]
	if !exists {
		bucket = &RateBucket{
			Tokens:     tier.MaxTokens,
			LastUpdate: now,
		}
		rls.Lobby.httpRateLimits[bucketKey] = bucket
	}

	// Leak tokens based on elapsed time
	elapsed := now.Sub(bucket.LastUpdate).Seconds()
	bucket.Tokens += elapsed * tier.RefillRate
	if bucket.Tokens > tier.MaxTokens {
		bucket.Tokens = tier.MaxTokens
	}
	bucket.LastUpdate = now

	if bucket.Tokens < 1.0 {
		rls.Mu.Unlock()
		deficit := 1.0 - bucket.Tokens
		retryAfter := deficit / tier.RefillRate
		return false, math.Ceil(retryAfter)
	}

	bucket.Tokens -= 1.0
	// Track within sliding window
	windowKey := tier.Name + "_" + key
	if _, exists := rls.Lobby.httpRateLimits[windowKey]; !exists {
		rls.Lobby.httpRateLimits[windowKey] = &RateBucket{
			Tokens:     1,
			LastUpdate: now,
		}
	} else {
		rls.Lobby.httpRateLimits[windowKey].Tokens += 1
	}
	rls.Mu.Unlock()

	return true, 0
}

// Middleware wraps an HTTP handler with rate limiting (Task 3102).
func (rls *RateLimiterService) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := rls.Allow(r)
		if !allowed {
			// Task 3106: Report to Prometheus
			rateLimitedCounter.Inc()

			tierKey, _ := rls.resolveKey(r)
			tier := rls.resolveTier(tierKey)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", formatRetryAfter(retryAfter))
			w.Header().Set("X-RateLimit-Limit", strconv.FormatFloat(tier.MaxTokens, 'f', 0, 64))
			http.Error(w, `{"error":"rate_limit_exceeded","retry_after":`+formatRetryAfter(retryAfter)+`}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WithRateLimit returns an http.HandlerFunc that applies rate limiting for a specific tier.
// It wraps the provided handler function with the tier's quota constraints.
func (rls *RateLimiterService) WithRateLimit(handlerFunc http.HandlerFunc, tierName string) http.HandlerFunc {
	// Normalize tier name to a key format used by resolveTier
	tierKey := "tier:" + tierName

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := rls.Allow(r)
		if !allowed {
			rateLimitedCounter.Inc()
			tier := rls.resolveTier(tierKey)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", formatRetryAfter(retryAfter))
			w.Header().Set("X-RateLimit-Limit", strconv.FormatFloat(tier.MaxTokens, 'f', 0, 64))
			http.Error(w, `{"error":"rate_limit_exceeded","retry_after":`+formatRetryAfter(retryAfter)+`}`, http.StatusTooManyRequests)
			return
		}
		handlerFunc.ServeHTTP(w, r)
	})
}

// formatRetryAfter converts seconds to a human-readable retry-after value.
func formatRetryAfter(seconds float64) string {
	if seconds < 1 {
		return "1"
	}
	return strconv.FormatInt(int64(math.Ceil(seconds)), 10)
}

// CleanupStaleEntries removes expired rate limit entries periodically.
func (rls *RateLimiterService) CleanupStaleEntries(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			rls.cleanupLocked()
		}
	}()
}

func (rls *RateLimiterService) cleanupLocked() {
	rls.Mu.Lock()
	defer rls.Mu.Unlock()

	now := time.Now()
	cleanupInterval := 5 * time.Minute

	for key, bucket := range rls.Lobby.httpRateLimits {
		if now.Sub(bucket.LastUpdate) > cleanupInterval {
			delete(rls.Lobby.httpRateLimits, key)
		}
	}
}

// GetStatus returns the current rate limit status for a wallet.
func (rls *RateLimiterService) GetStatus(wallet string) map[string]interface{} {
	wallet = strings.ToLower(strings.TrimSpace(wallet))

	rls.Mu.RLock()
	defer rls.Mu.RUnlock()

	status := make(map[string]interface{})

	if q, exists := rls.WalletQuotas[wallet]; exists {
		status["tier"] = q.TierName
		status["is_admin"] = q.IsAdmin
	} else if rls.AdminWallets[wallet] {
		status["tier"] = "admin_bypass"
		status["is_admin"] = true
	} else {
		status["tier"] = "default"
		status["is_admin"] = false
	}

	return status
}
