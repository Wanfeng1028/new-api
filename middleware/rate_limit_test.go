package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func useRateLimitMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	previousRedisEnabled := common.RedisEnabled
	previousRedisClient := common.RDB
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	require.NoError(t, redisClient.Ping(context.Background()).Err())

	common.RedisEnabled = true
	common.RDB = redisClient
	t.Cleanup(func() {
		_ = redisClient.Close()
		common.RedisEnabled = previousRedisEnabled
		common.RDB = previousRedisClient
	})

	return redisServer, redisClient
}

func performRateLimitRequest(router http.Handler, path string, remoteAddr string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = remoteAddr
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestRedisIPRateLimiterThresholdTTLAndNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/limited", rateLimitFactory(2, 37, "TEST"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	remoteAddr := "192.0.2.10:12345"
	legacyKey := "rateLimit:TEST192.0.2.10"
	_, err := redisServer.Push(legacyKey, "legacy-list-entry")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/limited", remoteAddr).Code)
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/limited", remoteAddr).Code)
	limitedResponse := performRateLimitRequest(router, "/limited", remoteAddr)
	assert.Equal(t, http.StatusTooManyRequests, limitedResponse.Code)
	assert.Equal(t, "37", limitedResponse.Header().Get("Retry-After"))

	key := redisIPRateLimitKey("TEST", "192.0.2.10")
	count, err := redisServer.Get(key)
	require.NoError(t, err)
	assert.Equal(t, "3", count)
	assert.Equal(t, 37*time.Second, redisServer.TTL(key))
	assert.True(t, redisServer.Exists(legacyKey), "the v2 counter must not touch an old list key")
}

func TestRedisUserRateLimiterUsesSharedFixedWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	router := gin.New()
	router.GET(
		"/limited",
		func(c *gin.Context) { c.Set("id", 42) },
		userRateLimitFactory(1, 23, "USER", true),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)

	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/limited", "192.0.2.20:12345").Code)
	assert.Equal(t, http.StatusTooManyRequests, performRateLimitRequest(router, "/limited", "198.51.100.20:12345").Code)

	key := redisUserRateLimitKey("USER", 42)
	assert.True(t, redisServer.Exists(key))
	assert.Equal(t, 23*time.Second, redisServer.TTL(key))
}

// Identity-bound (per-user) rate limits skip admins and above, whichever auth
// path populated the role: dashboard sessions set the "role" key, token auth
// populates constant.ContextKeyUserRole from the user record. Regular users
// keep hitting the same limits on both storage backends.
func TestUserRateLimitExemptsAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	buildRouter := func(mark string, identity func(c *gin.Context)) *gin.Engine {
		router := gin.New()
		router.GET("/limited", identity, userRateLimitFactory(1, 60, mark, true), func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		return router
	}
	roleKeyIdentity := func(id, role int) func(c *gin.Context) {
		return func(c *gin.Context) {
			c.Set("id", id)
			c.Set("role", role)
		}
	}
	typedKeyIdentity := func(id, role int) func(c *gin.Context) {
		return func(c *gin.Context) {
			c.Set("id", id)
			common.SetContextKey(c, constant.ContextKeyUserRole, role)
		}
	}

	for _, tc := range []struct {
		name     string
		backend  string
		identity func(id, role int) func(c *gin.Context)
	}{
		{name: "redis/dashboard role key", backend: "redis", identity: roleKeyIdentity},
		{name: "redis/token role key", backend: "redis", identity: typedKeyIdentity},
		{name: "memory/dashboard role key", backend: "memory", identity: roleKeyIdentity},
		{name: "memory/token role key", backend: "memory", identity: typedKeyIdentity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == "redis" {
				useRateLimitMiniRedis(t)
			} else {
				previousRedisEnabled := common.RedisEnabled
				common.RedisEnabled = false
				t.Cleanup(func() { common.RedisEnabled = previousRedisEnabled })
			}
			mark := "EXEMPT-" + tc.name

			commonRouter := buildRouter(mark, tc.identity(930001, common.RoleCommonUser))
			assert.Equal(t, http.StatusNoContent, performRateLimitRequest(commonRouter, "/limited", "192.0.2.80:12345").Code)
			assert.Equal(t, http.StatusTooManyRequests, performRateLimitRequest(commonRouter, "/limited", "192.0.2.80:12345").Code)

			for _, role := range []int{common.RoleAdminUser, common.RoleRootUser} {
				adminRouter := buildRouter(mark, tc.identity(930002+role, role))
				for range 3 {
					assert.Equal(t, http.StatusNoContent, performRateLimitRequest(adminRouter, "/limited", "192.0.2.81:12345").Code, "role %d must not be rate limited", role)
				}
			}
		})
	}
}

// UserCriticalRateLimit guards account-security endpoints (access tokens,
// 2FA/passkey, security verification, email binding, invite quota transfer),
// so it opts out of the admin exemption: admins and root users are throttled
// exactly like regular users, on both storage backends and both role sources.
func TestUserCriticalRateLimitThrottlesAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousEnable, previousNum, previousDuration := common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration
	t.Cleanup(func() {
		common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = previousEnable, previousNum, previousDuration
	})
	common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = true, 1, 60

	roleKeyIdentity := func(id, role int) func(c *gin.Context) {
		return func(c *gin.Context) {
			c.Set("id", id)
			c.Set("role", role)
		}
	}
	typedKeyIdentity := func(id, role int) func(c *gin.Context) {
		return func(c *gin.Context) {
			c.Set("id", id)
			common.SetContextKey(c, constant.ContextKeyUserRole, role)
		}
	}

	for _, tc := range []struct {
		name     string
		backend  string
		identity func(id, role int) func(c *gin.Context)
	}{
		{name: "redis/dashboard role key", backend: "redis", identity: roleKeyIdentity},
		{name: "redis/token role key", backend: "redis", identity: typedKeyIdentity},
		{name: "memory/dashboard role key", backend: "memory", identity: roleKeyIdentity},
		{name: "memory/token role key", backend: "memory", identity: typedKeyIdentity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == "redis" {
				useRateLimitMiniRedis(t)
			} else {
				previousRedisEnabled := common.RedisEnabled
				common.RedisEnabled = false
				t.Cleanup(func() { common.RedisEnabled = previousRedisEnabled })
			}
			scope := "test-" + tc.name

			for _, role := range []int{common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser} {
				router := gin.New()
				router.GET("/limited", tc.identity(940000+role, role), UserCriticalRateLimit(scope), func(c *gin.Context) {
					c.Status(http.StatusNoContent)
				})
				assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/limited", "192.0.2.82:12345").Code)
				assert.Equal(t, http.StatusTooManyRequests, performRateLimitRequest(router, "/limited", "192.0.2.82:12345").Code, "role %d must stay throttled on security endpoints", role)
			}
		})
	}
}

// SearchRateLimit keeps the admin exemption: only the security-endpoint
// limiter (UserCriticalRateLimit) opts out of it.
func TestSearchRateLimitStillExemptsAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousEnable, previousNum, previousDuration := common.SearchRateLimitEnable, common.SearchRateLimitNum, common.SearchRateLimitDuration
	t.Cleanup(func() {
		common.SearchRateLimitEnable, common.SearchRateLimitNum, common.SearchRateLimitDuration = previousEnable, previousNum, previousDuration
	})
	common.SearchRateLimitEnable, common.SearchRateLimitNum, common.SearchRateLimitDuration = true, 1, 60

	for _, tc := range []struct {
		name    string
		backend string
		baseID  int
	}{
		{name: "redis", backend: "redis", baseID: 950000},
		{name: "memory", backend: "memory", baseID: 960000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.backend == "redis" {
				useRateLimitMiniRedis(t)
			} else {
				previousRedisEnabled := common.RedisEnabled
				common.RedisEnabled = false
				t.Cleanup(func() { common.RedisEnabled = previousRedisEnabled })
			}

			commonRouter := gin.New()
			commonRouter.GET("/search", func(c *gin.Context) {
				c.Set("id", tc.baseID+int(common.RoleCommonUser))
				c.Set("role", common.RoleCommonUser)
			}, SearchRateLimit(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			assert.Equal(t, http.StatusNoContent, performRateLimitRequest(commonRouter, "/search", "192.0.2.83:12345").Code)
			assert.Equal(t, http.StatusTooManyRequests, performRateLimitRequest(commonRouter, "/search", "192.0.2.83:12345").Code)

			adminRouter := gin.New()
			adminRouter.GET("/search", func(c *gin.Context) {
				c.Set("id", tc.baseID+int(common.RoleRootUser))
				c.Set("role", common.RoleRootUser)
			}, SearchRateLimit(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			for range 3 {
				assert.Equal(t, http.StatusNoContent, performRateLimitRequest(adminRouter, "/search", "192.0.2.84:12345").Code, "admins must stay exempt from search rate limiting")
			}
		})
	}
}

func TestRedisEmailVerificationRateLimiterPreservesResponseAndTTL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisServer, _ := useRateLimitMiniRedis(t)

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/verify", EmailVerificationRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	remoteAddr := "192.0.2.30:12345"
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/verify", remoteAddr).Code)
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/verify", remoteAddr).Code)
	response := performRateLimitRequest(router, "/verify", remoteAddr)
	assert.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.JSONEq(t, `{"success":false,"message":"Please wait 30 seconds before requesting another verification code.","message_key":"Please wait {{seconds}} seconds before requesting another verification code.","message_params":{"seconds":30}}`, response.Body.String())

	key := redisIPRateLimitKey(EmailVerificationRateLimitMark, "192.0.2.30")
	assert.True(t, redisServer.Exists(key))
	assert.Equal(t, time.Duration(EmailVerificationDuration)*time.Second, redisServer.TTL(key))
}

func TestRedisFixedWindowIsAtomicUnderConcurrency(t *testing.T) {
	redisServer, _ := useRateLimitMiniRedis(t)
	const (
		requestCount = 20
		maximumCount = 7
		duration     = int64(41)
	)
	key := redisIPRateLimitKey("CONCURRENT", "192.0.2.40")

	var allowedCount atomic.Int64
	errorsFound := make(chan error, requestCount)
	var waitGroup sync.WaitGroup
	waitGroup.Add(requestCount)
	for range requestCount {
		go func() {
			defer waitGroup.Done()
			allowed, _, _, err := redisFixedWindowTake(context.Background(), key, maximumCount, duration)
			if err != nil {
				errorsFound <- err
				return
			}
			if allowed {
				allowedCount.Add(1)
			}
		}()
	}
	waitGroup.Wait()
	close(errorsFound)
	for err := range errorsFound {
		require.NoError(t, err)
	}

	assert.Equal(t, int64(maximumCount), allowedCount.Load())
	count, err := redisServer.Get(key)
	require.NoError(t, err)
	assert.Equal(t, "20", count)
	assert.Equal(t, time.Duration(duration)*time.Second, redisServer.TTL(key))
}

func TestRedisFixedWindowResetsAtBoundary(t *testing.T) {
	redisServer, _ := useRateLimitMiniRedis(t)
	const duration = int64(10)
	key := redisIPRateLimitKey("BOUNDARY", "192.0.2.50")

	for range 2 {
		allowed, _, _, err := redisFixedWindowTake(context.Background(), key, 2, duration)
		require.NoError(t, err)
		assert.True(t, allowed)
	}
	allowed, _, _, err := redisFixedWindowTake(context.Background(), key, 2, duration)
	require.NoError(t, err)
	assert.False(t, allowed)

	// This reset is intentional fixed-window behavior. A client can consume one
	// full allowance immediately before and another immediately after a boundary.
	redisServer.FastForward(time.Duration(duration) * time.Second)
	for range 2 {
		allowed, _, _, err = redisFixedWindowTake(context.Background(), key, 2, duration)
		require.NoError(t, err)
		assert.True(t, allowed)
	}
}

func TestRedisFixedWindowRepairsCounterWithoutTTL(t *testing.T) {
	redisServer, _ := useRateLimitMiniRedis(t)
	const duration = int64(29)
	key := redisIPRateLimitKey("MISSING-TTL", "192.0.2.51")
	redisServer.Set(key, "5")

	allowed, count, ttl, err := redisFixedWindowTake(context.Background(), key, 3, duration)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(6), count)
	assert.Equal(t, duration, ttl)
	assert.Equal(t, time.Duration(duration)*time.Second, redisServer.TTL(key))

	redisServer.FastForward(time.Duration(duration) * time.Second)
	assert.False(t, redisServer.Exists(key), "a recovered counter must not remain permanently rate-limited")
}

func TestRedisFailurePolicies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, redisClient := useRateLimitMiniRedis(t)
	require.NoError(t, redisClient.Close())

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	router.GET("/ip", rateLimitFactory(1, 30, "FAIL-IP"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET(
		"/user",
		func(c *gin.Context) { c.Set("id", 7) },
		userRateLimitFactory(1, 30, "FAIL-USER", true),
		func(c *gin.Context) { c.Status(http.StatusNoContent) },
	)
	router.GET("/email", EmailVerificationRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	ipResponse := performRateLimitRequest(router, "/ip", "192.0.2.60:12345")
	assert.Equal(t, http.StatusInternalServerError, ipResponse.Code)
	assert.Empty(t, ipResponse.Body.String())
	userResponse := performRateLimitRequest(router, "/user", "192.0.2.61:12345")
	assert.Equal(t, http.StatusInternalServerError, userResponse.Code)
	assert.Empty(t, userResponse.Body.String())
	assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/email", "192.0.2.62:12345").Code)
}

func TestGlobalWebRateLimitCountsFrontendFilesInStaticBucket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousWebEnable, previousWebNum, previousWebDuration := common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration
	previousStaticEnable, previousStaticNum, previousStaticDuration := common.GlobalStaticRateLimitEnable, common.GlobalStaticRateLimitNum, common.GlobalStaticRateLimitDuration
	t.Cleanup(func() {
		common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration = previousWebEnable, previousWebNum, previousWebDuration
		common.GlobalStaticRateLimitEnable, common.GlobalStaticRateLimitNum, common.GlobalStaticRateLimitDuration = previousStaticEnable, previousStaticNum, previousStaticDuration
	})
	common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration = true, 2, 41
	common.GlobalStaticRateLimitNum, common.GlobalStaticRateLimitDuration = 2, 43

	frontendDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(frontendDir, "static", "js"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(frontendDir, "static", "js", "app.js"), []byte("app"), 0o644))
	frontendFS := static.LocalFile(frontendDir, false)

	const clientIP = "192.0.2.70"
	const remoteAddr = clientIP + ":12345"

	t.Run("static limiter disabled by default", func(t *testing.T) {
		redisServer, _ := useRateLimitMiniRedis(t)
		common.GlobalStaticRateLimitEnable = false
		router := gin.New()
		require.NoError(t, router.SetTrustedProxies(nil))
		router.NoRoute(GlobalWebRateLimit(frontendFS), func(c *gin.Context) { c.Status(http.StatusNoContent) })

		for range 5 {
			assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/static/js/app.js", remoteAddr).Code)
		}
		assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/channels", remoteAddr).Code)
		assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/static/js/missing.js", remoteAddr).Code)
		pageResponse := performRateLimitRequest(router, "/channels", remoteAddr)
		assert.Equal(t, http.StatusTooManyRequests, pageResponse.Code)
		assert.Equal(t, "41", pageResponse.Header().Get("Retry-After"))

		webCount, err := redisServer.Get(redisIPRateLimitKey("GW", clientIP))
		require.NoError(t, err)
		assert.Equal(t, "3", webCount)
		assert.False(t, redisServer.Exists(redisIPRateLimitKey("GS", clientIP)))
	})

	t.Run("static limiter enabled uses its own bucket", func(t *testing.T) {
		redisServer, _ := useRateLimitMiniRedis(t)
		common.GlobalStaticRateLimitEnable = true
		router := gin.New()
		require.NoError(t, router.SetTrustedProxies(nil))
		router.NoRoute(GlobalWebRateLimit(frontendFS), func(c *gin.Context) { c.Status(http.StatusNoContent) })

		assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/static/js/app.js", remoteAddr).Code)
		assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/static/js/app.js", remoteAddr).Code)
		staticResponse := performRateLimitRequest(router, "/static/js/app.js", remoteAddr)
		assert.Equal(t, http.StatusTooManyRequests, staticResponse.Code)
		assert.Equal(t, "43", staticResponse.Header().Get("Retry-After"))
		assert.Equal(t, http.StatusNoContent, performRateLimitRequest(router, "/channels", remoteAddr).Code)

		staticCount, err := redisServer.Get(redisIPRateLimitKey("GS", clientIP))
		require.NoError(t, err)
		assert.Equal(t, "3", staticCount)
		webCount, err := redisServer.Get(redisIPRateLimitKey("GW", clientIP))
		require.NoError(t, err)
		assert.Equal(t, "1", webCount)
	})
}
