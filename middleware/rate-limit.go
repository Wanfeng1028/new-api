package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

const redisRateLimitNamespace = "rateLimit:v2"

// Redis rate limiting intentionally uses a fixed window. The single Lua script
// makes increment, expiry, and the limit decision atomic, while retaining the
// simple fixed-window behavior: traffic at a window boundary can burst up to
// twice the configured limit. Do not replace this with a sliding-window ZSET
// unless that externally visible behavior is intentionally changed.
const redisFixedWindowScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
local ttl = redis.call('TTL', KEYS[1])
if ttl < 0 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
  ttl = redis.call('TTL', KEYS[1])
end
if count > tonumber(ARGV[1]) then
  return {0, count, ttl}
end
return {1, count, ttl}
`

var inMemoryRateLimiter common.InMemoryRateLimiter

var defNext = func(c *gin.Context) {
	c.Next()
}

func redisIPRateLimitKey(mark string, clientIP string) string {
	return fmt.Sprintf("%s:ip:%s:%s", redisRateLimitNamespace, mark, clientIP)
}

func redisUserRateLimitKey(mark string, userID int) string {
	return fmt.Sprintf("%s:user:%s:%d", redisRateLimitNamespace, mark, userID)
}

func redisReplyInteger(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected Redis integer reply type %T", value)
	}
}

func redisFixedWindowTake(ctx context.Context, key string, maxRequestNum int, duration int64) (bool, int64, int64, error) {
	if common.RDB == nil {
		return false, 0, 0, errors.New("Redis client is not initialized")
	}
	if key == "" {
		return false, 0, 0, errors.New("rate limit key is empty")
	}
	if maxRequestNum <= 0 {
		return false, 0, 0, errors.New("rate limit maximum must be positive")
	}
	if duration <= 0 {
		return false, 0, 0, errors.New("rate limit duration must be positive")
	}

	values, err := common.RDB.Eval(
		ctx,
		redisFixedWindowScript,
		[]string{key},
		maxRequestNum,
		duration,
	).Slice()
	if err != nil {
		return false, 0, 0, err
	}
	if len(values) != 3 {
		return false, 0, 0, fmt.Errorf("unexpected Redis rate limit reply length %d", len(values))
	}

	allowedValue, err := redisReplyInteger(values[0])
	if err != nil {
		return false, 0, 0, err
	}
	count, err := redisReplyInteger(values[1])
	if err != nil {
		return false, 0, 0, err
	}
	ttlSeconds, err := redisReplyInteger(values[2])
	if err != nil {
		return false, 0, 0, err
	}

	return allowedValue == 1, count, ttlSeconds, nil
}

func redisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	allowed, _, ttlSeconds, err := redisFixedWindowTake(
		c.Request.Context(),
		redisIPRateLimitKey(mark, c.ClientIP()),
		maxRequestNum,
		duration,
	)
	if err != nil {
		// Review F-15: a Redis outage must not fail closed with a bare
		// 500 while the in-memory implementation would have served the
		// request. Degrade to the in-memory limiter (same key shape as
		// memoryRateLimiter) and log, so both implementations behave
		// alike when their store is unavailable.
		logger.LogError(c.Request.Context(), common.LogText("rate limit check failed (mark=%s), falling back to in-memory limiter: %v", mark, err))
		memoryRateLimitFallback(c, mark+c.ClientIP(), maxRequestNum, duration)
		return
	}
	if !allowed {
		writeRateLimited(c, ttlSeconds)
	}
}

// memoryRateLimitFallback applies the in-memory limiter for callers whose
// primary (Redis) limiter is unavailable. It is the shared degradation
// path for every Redis-backed limiter in this file (review F-15).
func memoryRateLimitFallback(c *gin.Context, key string, maxRequestNum int, duration int64) {
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		writeRateLimited(c, duration)
	}
}

func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	key := mark + c.ClientIP()
	if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		writeRateLimited(c, duration)
		return
	}
}

// writeRateLimited rejects the request with 429 and a Retry-After hint so
// clients can back off instead of treating the rejection as a fatal error.
// The in-memory limiter cannot report the remaining window, so callers
// without a TTL pass the full window duration as a conservative upper bound.
// Review F-01 (residual): the response also carries a JSON business body
// (success=false + message), because a bare 429 on the login path was
// rendered by the frontend as a generic credential failure ("password
// incorrect") during the 2026-10-24 lockout incident. The status code is
// unchanged, so existing callers that branch on 429 are unaffected.
func writeRateLimited(c *gin.Context, retryAfterSeconds int64) {
	if retryAfterSeconds > 0 {
		c.Header("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
	}
	c.JSON(http.StatusTooManyRequests, gin.H{
		"success": false,
		"message": "请求过于频繁，请稍后再试",
	})
	c.Abort()
}

func rateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			redisRateLimiter(c, maxRequestNum, duration, mark)
		}
	}
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		memoryRateLimiter(c, maxRequestNum, duration, mark)
	}
}

// GlobalWebRateLimit limits dashboard pages and other fallback paths per IP.
// Files that exist in the frontend build count against a separate static
// limiter instead, so lazy-loaded chunks and icons do not use up the page
// budget. The static limiter is disabled by default.
func GlobalWebRateLimit(frontendFS static.ServeFileSystem) func(c *gin.Context) {
	webLimit := defNext
	if common.GlobalWebRateLimitEnable {
		webLimit = rateLimitFactory(common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration, "GW")
	}
	staticLimit := defNext
	if common.GlobalStaticRateLimitEnable {
		staticLimit = rateLimitFactory(common.GlobalStaticRateLimitNum, common.GlobalStaticRateLimitDuration, "GS")
	}
	return func(c *gin.Context) {
		if frontendFS.Exists("/", c.Request.URL.Path) {
			staticLimit(c)
			return
		}
		webLimit(c)
	}
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	if common.GlobalApiRateLimitEnable {
		return rateLimitFactory(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration, "GA")
	}
	return defNext
}

func CriticalRateLimit() func(c *gin.Context) {
	if common.CriticalRateLimitEnable {
		return rateLimitFactory(common.CriticalRateLimitNum, common.CriticalRateLimitDuration, "CT")
	}
	return defNext
}

// RefreshRateLimit gives session-token refresh its own generous bucket.
// Refresh is cookie-authenticated and rotates the refresh token; the
// marketing site's session bridge and every open console tab call it on
// load. Sharing the 20-per-20min critical bucket with password login let
// a burst of refreshes burn the whole budget and lock the site owner out
// of password login with bare 429s (incident 2026-10-24). 600/20min per
// IP is far above any legitimate multi-tab pattern while still capping
// scripted abuse.
func RefreshRateLimit() func(c *gin.Context) {
	if common.CriticalRateLimitEnable {
		return rateLimitFactory(600, common.CriticalRateLimitDuration, "RF")
	}
	return defNext
}

// loginAdminCache memoizes username -> is-admin lookups for
// LoginRateLimit so the login path does not hit the DB on every attempt.
var loginAdminCache sync.Map // string -> loginAdminEntry

type loginAdminEntry struct {
	admin bool
	exp   time.Time
}

// isAdminUsername reports whether username belongs to an admin-or-above
// account. Fails closed (false) on any error so a DB hiccup never widens
// anyone's allowance.
func isAdminUsername(username string) bool {
	if username == "" || model.DB == nil {
		return false
	}
	if v, ok := loginAdminCache.Load(username); ok {
		if e, ok := v.(loginAdminEntry); ok && time.Now().Before(e.exp) {
			return e.admin
		}
	}
	var probe model.User
	err := model.DB.Select("role").Where("username = ?", username).First(&probe).Error
	admin := err == nil && probe.Role >= common.RoleAdminUser
	loginAdminCache.Store(username, loginAdminEntry{admin: admin, exp: time.Now().Add(time.Minute)})
	return admin
}

// peekLoginUsername extracts the username from a JSON login body without
// consuming it: the body is restored for the downstream handler.
func peekLoginUsername(c *gin.Context) string {
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 64*1024))
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	var payload struct {
		Username string `json:"username"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	return payload.Username
}

func loginRateLimiter(c *gin.Context, key string, maxRequestNum int, duration int64) {
	if common.RedisEnabled {
		allowed, _, ttlSeconds, err := redisFixedWindowTake(
			c.Request.Context(),
			redisRateLimitNamespace+":LG:"+key,
			maxRequestNum,
			duration,
		)
		if err != nil {
			// Review F-15: degrade to the in-memory limiter instead of
			// failing closed with a 500 (see redisRateLimiter).
			logger.LogError(c.Request.Context(), common.LogText("login rate limit check failed, falling back to in-memory limiter: %v", err))
			memoryRateLimitFallback(c, "LG:"+key, maxRequestNum, duration)
			return
		}
		if !allowed {
			writeRateLimited(c, ttlSeconds)
		}
		return
	}
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	if !inMemoryRateLimiter.Request("LG:"+key, maxRequestNum, duration) {
		writeRateLimited(c, duration)
	}
}

// LoginRateLimit throttles password login per client IP + username
// instead of per IP alone. The old shared critical bucket keyed only by
// IP (behind Cloudflare, an edge IP shared by many visitors), so other
// critical-route traffic or another account's attempts could exhaust a
// user's login budget. Admin accounts get a far larger allowance: the
// site owner must never be throttled out of his own console, while the
// per-IP+account cap, the pair lockout for regular users, 2FA and the
// new-IP admin login alerts still bound scripted guessing.
func LoginRateLimit() func(c *gin.Context) {
	if !common.CriticalRateLimitEnable {
		return defNext
	}
	return func(c *gin.Context) {
		username := peekLoginUsername(c)
		maxRequestNum := common.CriticalRateLimitNum
		if isAdminUsername(username) {
			maxRequestNum = 300
		}
		loginRateLimiter(c, c.ClientIP()+":"+username, maxRequestNum, common.CriticalRateLimitDuration)
	}
}

func UserCriticalRateLimit(scope string) func(c *gin.Context) {
	if !common.CriticalRateLimitEnable {
		return defNext
	}
	// Admin exemption is disabled here on purpose: this limiter guards
	// account-security endpoints (access tokens, 2FA/passkey, security
	// verification, email binding, invite quota transfer), where throttling
	// must keep protecting admins too, e.g. if a session is hijacked.
	return userRateLimitFactory(
		common.CriticalRateLimitNum,
		common.CriticalRateLimitDuration,
		"UC:"+scope,
		false,
	)
}

func DownloadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.DownloadRateLimitNum, common.DownloadRateLimitDuration, "DW")
}

func UploadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.UploadRateLimitNum, common.UploadRateLimitDuration, "UP")
}

// adminRateLimitExempt reports whether the authenticated user holds an admin
// role or above. Identity-bound rate limits that opt in (the per-user limiters
// built by userRateLimitFactory with exemptAdmins set, and the model request
// rate limit) skip those accounts: operators must not be throttled by limits
// meant for regular users, including
// when calling models with their own API tokens. IP-bound limiters run before
// authentication and are unaffected. The role is read from server-side auth
// state only: the "role" key set for dashboard sessions and
// constant.ContextKeyUserRole populated from the user record by token auth.
func adminRateLimitExempt(c *gin.Context) bool {
	if c.GetInt("role") >= common.RoleAdminUser {
		return true
	}
	return common.GetContextKeyInt(c, constant.ContextKeyUserRole) >= common.RoleAdminUser
}

// userRateLimitFactory creates a rate limiter keyed by authenticated user ID
// instead of client IP, making it resistant to proxy rotation attacks.
// Must be used AFTER authentication middleware (UserAuth). exemptAdmins
// controls whether admin-and-above accounts skip the limiter; limiters
// guarding account-security endpoints pass false so admins stay throttled.
func userRateLimitFactory(maxRequestNum int, duration int64, mark string, exemptAdmins bool) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			userID := c.GetInt("id")
			if userID == 0 {
				c.Status(http.StatusUnauthorized)
				c.Abort()
				return
			}
			if exemptAdmins && adminRateLimitExempt(c) {
				return
			}
			userRedisRateLimiter(c, maxRequestNum, duration, redisUserRateLimitKey(mark, userID))
		}
	}
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		userID := c.GetInt("id")
		if userID == 0 {
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}
		if exemptAdmins && adminRateLimitExempt(c) {
			return
		}
		key := fmt.Sprintf("%s:user:%d", mark, userID)
		if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
			writeRateLimited(c, duration)
			return
		}
	}
}

// userRedisRateLimiter is like redisRateLimiter but accepts a pre-built key
// (to support user-ID-based keys).
func userRedisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, key string) {
	allowed, _, ttlSeconds, err := redisFixedWindowTake(c.Request.Context(), key, maxRequestNum, duration)
	if err != nil {
		// Review F-15: degrade to the in-memory limiter instead of
		// failing closed with a 500 (see redisRateLimiter). The Redis
		// key already encodes the user identity, so it doubles as the
		// in-memory key.
		logger.LogError(c.Request.Context(), common.LogText("rate limit check failed (key=%s), falling back to in-memory limiter: %v", key, err))
		memoryRateLimitFallback(c, key, maxRequestNum, duration)
		return
	}
	if !allowed {
		writeRateLimited(c, ttlSeconds)
	}
}

// SearchRateLimit returns a per-user rate limiter for search endpoints.
// Configurable via SEARCH_RATE_LIMIT_ENABLE / SEARCH_RATE_LIMIT / SEARCH_RATE_LIMIT_DURATION.
func SearchRateLimit() func(c *gin.Context) {
	if !common.SearchRateLimitEnable {
		return defNext
	}
	return userRateLimitFactory(common.SearchRateLimitNum, common.SearchRateLimitDuration, "SR", true)
}
