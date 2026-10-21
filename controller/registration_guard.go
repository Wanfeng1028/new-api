package controller

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// BotsRouter fork patch: one-account-per-network-and-device-per-year policy
// for self-service registration. Both public creation paths (password
// registration and OAuth sign-up) record the client IP / User-Agent on the
// user row and consult checkRegistrationLimit before creating an account.
// The test is deliberately "wide": a registration is refused only when the
// same network (IP) shows up again within a year together with the same
// device, where device identity is either the same User-Agent or the
// reg_mark cookie planted on the earlier registration. Admin-created
// accounts never pass through here.

const (
	regMarkCookieName = "reg_mark"
	// regLimitWindow is one year, in unix seconds.
	regLimitWindow    = int64(365 * 24 * 3600)
	regLimitMessage   = "同一网络与设备一年内只能注册一个账号，如有疑问请联系管理员。"
	regMarkCookieTTL  = 31536000 // one year, in seconds
	regUserAgentLimit = 512
)

// registrationClientIP resolves the client address behind the edge:
// Cloudflare's connecting-IP header wins, then the first X-Forwarded-For
// hop, then gin's own resolution.
func registrationClientIP(c *gin.Context) string {
	if ip := strings.TrimSpace(c.GetHeader("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		if first, _, _ := strings.Cut(xff, ","); strings.TrimSpace(first) != "" {
			return strings.TrimSpace(first)
		}
	}
	return c.ClientIP()
}

// registrationUserAgent returns the request User-Agent capped at the
// storage limit of the users.reg_ua column.
func registrationUserAgent(c *gin.Context) string {
	ua := c.Request.UserAgent()
	if len(ua) > regUserAgentLimit {
		ua = ua[:regUserAgentLimit]
	}
	return ua
}

// checkRegistrationLimit reports whether this request must be refused
// under the one-per-network-and-device-per-year policy.
func checkRegistrationLimit(c *gin.Context) (bool, error) {
	ip := registrationClientIP(c)
	if ip == "" {
		return false, nil
	}
	query := model.DB.Model(&model.User{}).
		Where("reg_ip = ? AND created_at >= ?", ip, time.Now().Unix()-regLimitWindow)
	if _, err := c.Cookie(regMarkCookieName); err != nil {
		// No device mark from an earlier registration: the same network
		// only counts together with the same User-Agent.
		query = query.Where("reg_ua = ?", registrationUserAgent(c))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// setRegMarkCookie plants the long-lived device mark on a successful
// registration response. The value is a random nonce; only its presence
// matters to checkRegistrationLimit. A failure to mint the nonce simply
// skips the mark and never fails the registration itself.
func setRegMarkCookie(c *gin.Context) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     regMarkCookieName,
		Value:    hex.EncodeToString(nonce),
		Domain:   "botsarena.tech",
		Path:     "/",
		MaxAge:   regMarkCookieTTL,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
