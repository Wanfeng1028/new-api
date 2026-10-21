package controller

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// BotsRouter fork patch: one-account-per-network-and-device-per-year policy
// for self-service registration. Both public creation paths (password
// registration and OAuth sign-up) record the client IP / User-Agent and a
// device-mark hash on the user row and consult checkRegistrationLimit
// before creating an account. A registration is refused when, inside the
// last year, either (a) the reg_mark cookie presented by the browser
// hashes to a mark stored by an earlier registration (same device), or
// (b) an earlier account shares both the client IP and a non-empty
// User-Agent (same network and device fingerprint). Admin-created
// accounts never pass through here.

const (
	regMarkCookieName = "reg_mark"
	// regLimitWindow is one year, in unix seconds.
	regLimitWindow    = int64(365 * 24 * 3600)
	regLimitMessage   = "同一网络与设备一年内只能注册一个账号，如有疑问请联系管理员。"
	regMarkCookieTTL  = 31536000 // one year, in seconds
	regUserAgentLimit = 512
)

// NormalizeEmailForRegistration canonicalizes an address for every
// registration-time decision (occupancy check, verification-code key,
// stored value), so a single mailbox cannot mint extra accounts through
// plus aliases (user+tag@ anywhere) or Gmail dot tricks
// (u.ser@gmail.com is the same mailbox as user@gmail.com). Steps: lower
// case; drop the '+' suffix of the local part for every domain; for
// gmail.com / googlemail.com also drop all dots from the local part and
// unify the domain to gmail.com. Other domains keep their dots, which
// are significant there. Non-registration flows (password reset, bound
// mailbox changes) deliberately do not use this and keep exact matching.
func NormalizeEmailForRegistration(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	local, domain, found := strings.Cut(email, "@")
	if !found {
		return email
	}
	if plus := strings.IndexByte(local, '+'); plus >= 0 {
		local = local[:plus]
	}
	if domain == "gmail.com" || domain == "googlemail.com" {
		local = strings.ReplaceAll(local, ".", "")
		domain = "gmail.com"
	}
	return local + "@" + domain
}

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
// under the one-per-network-and-device-per-year policy: rule (a) matches
// the device mark, rule (b) matches network plus User-Agent.
func checkRegistrationLimit(c *gin.Context) (bool, error) {
	since := time.Now().Unix() - regLimitWindow
	ip := registrationClientIP(c)
	// Rule (a): the browser presents a reg_mark cookie minted by an
	// earlier registration; the users table stores only its hash.
	if mark, err := c.Cookie(regMarkCookieName); err == nil && mark != "" {
		var count int64
		if err := model.DB.Model(&model.User{}).
			Where("reg_mark = ? AND created_at >= ?", hashRegMark(mark), since).
			Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			common.RecordRegistrationBlock(ip)
			return true, nil
		}
	}
	// Rule (b): same network and same device fingerprint; an empty
	// User-Agent proves nothing, so both parts must be present.
	ua := registrationUserAgent(c)
	if ip != "" && ua != "" {
		var count int64
		if err := model.DB.Model(&model.User{}).
			Where("reg_ip = ? AND reg_ua = ? AND created_at >= ?", ip, ua, since).
			Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			common.RecordRegistrationBlock(ip)
			return true, nil
		}
	}
	return false, nil
}

// mintRegMark creates a fresh device-mark nonce and its stored form.
// The browser carries the nonce in the reg_mark cookie; the users table
// keeps only the SHA-256 hash, so a database leak does not hand out
// usable device marks. A crypto failure yields empty strings, which
// simply leaves the new account without a mark.
func mintRegMark() (nonce string, hash string) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", ""
	}
	nonce = hex.EncodeToString(buf)
	return nonce, hashRegMark(nonce)
}

func hashRegMark(nonce string) string {
	sum := sha256.Sum256([]byte(nonce))
	return hex.EncodeToString(sum[:])
}

// setRegMarkCookie plants the long-lived device mark on a successful
// registration response. The caller mints the nonce (see mintRegMark)
// so it can store the matching hash on the new user row; an empty nonce
// skips the mark and never fails the registration itself.
func setRegMarkCookie(c *gin.Context, nonce string) {
	if nonce == "" {
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     regMarkCookieName,
		Value:    nonce,
		Domain:   "botsarena.tech",
		Path:     "/",
		MaxAge:   regMarkCookieTTL,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
