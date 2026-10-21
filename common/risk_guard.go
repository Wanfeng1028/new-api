package common

import (
	"fmt"
	"sync"
	"time"
)

// BotsRouter fork patch: lightweight in-process risk controls for this
// single-instance deployment. All state is memory-only by design (the
// task explicitly accepts process state); a restart resets counters and
// the admin IP baseline, which fails open for alerts but never blocks
// legitimate logins beyond the short lockout window.

// RiskAlertEmail receives the L2 anomaly alerts.
const RiskAlertEmail = "wanfeng572@gmail.com"

// ---------- L1: password login lockout ----------

const (
	loginFailWindow   = 10 * time.Minute
	loginFailLimit    = 5
	loginLockDuration = 15 * time.Minute
)

type loginGuardState struct {
	failures    []time.Time
	lockedUntil time.Time
}

var (
	loginGuardMu sync.Mutex
	loginGuards  = map[string]*loginGuardState{}
)

func loginGuardKey(username, ip string) string {
	return username + "|" + ip
}

// LoginLocked reports whether this username+IP pair is inside an active
// lockout, together with the remaining lock time.
func LoginLocked(username, ip string) (bool, time.Duration) {
	now := time.Now()
	loginGuardMu.Lock()
	defer loginGuardMu.Unlock()
	state, ok := loginGuards[loginGuardKey(username, ip)]
	if !ok {
		return false, 0
	}
	if now.Before(state.lockedUntil) {
		return true, time.Until(state.lockedUntil)
	}
	if len(state.failures) == 0 {
		delete(loginGuards, loginGuardKey(username, ip))
	}
	return false, 0
}

// RecordLoginFailure notes one bad-password attempt for the pair and
// engages the lockout once the limit inside the window is reached.
func RecordLoginFailure(username, ip string) {
	now := time.Now()
	loginGuardMu.Lock()
	defer loginGuardMu.Unlock()
	key := loginGuardKey(username, ip)
	state, ok := loginGuards[key]
	if !ok {
		state = &loginGuardState{}
		loginGuards[key] = state
	}
	cutoff := now.Add(-loginFailWindow)
	kept := state.failures[:0]
	for _, at := range state.failures {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	state.failures = append(kept, now)
	if len(state.failures) >= loginFailLimit {
		state.lockedUntil = now.Add(loginLockDuration)
		state.failures = nil
	}
}

// RecordLoginSuccess clears the failure state of the pair.
func RecordLoginSuccess(username, ip string) {
	loginGuardMu.Lock()
	defer loginGuardMu.Unlock()
	delete(loginGuards, loginGuardKey(username, ip))
}

// ---------- L2: anomaly alert emails ----------

const (
	riskAlertInterval = time.Hour
	regBlockWindow    = time.Hour
	regBlockThreshold = 3
	topupFailWindow   = 10 * time.Minute
	topupFailLimit    = 5
	adminIPRetention  = 30 * 24 * time.Hour
)

var (
	riskMu           sync.Mutex
	riskLastAlert    = map[string]time.Time{}
	regBlockEvents   []time.Time
	topupFailEvents  = map[string][]time.Time{}
	adminSeenIPs     = map[string]map[string]time.Time{}
	adminIPBaselined = map[string]bool{}
)

// sendRiskAlert mails one alert unless the same category already alerted
// inside the throttle interval. It never blocks the caller.
func sendRiskAlert(category, subject, body string) {
	now := time.Now()
	riskMu.Lock()
	if last, ok := riskLastAlert[category]; ok && now.Sub(last) < riskAlertInterval {
		riskMu.Unlock()
		return
	}
	riskLastAlert[category] = now
	riskMu.Unlock()
	go func() {
		if err := SendEmail(subject, RiskAlertEmail, body); err != nil {
			SysLog(LogText("risk alert email failed (%s): %v", category, err))
		}
	}()
}

func alertBody(lines ...string) string {
	body := ""
	for _, line := range lines {
		body += "<p>" + line + "</p>"
	}
	return body
}

// NoteAdminLogin records a successful admin password login and alerts
// when the source IP was not seen for this admin in the last 30 days.
// The first login observed after process start only seeds the baseline
// so deploys do not produce a false alarm.
func NoteAdminLogin(username, ip string) {
	if ip == "" {
		return
	}
	now := time.Now()
	riskMu.Lock()
	seen, ok := adminSeenIPs[username]
	if !ok {
		seen = map[string]time.Time{}
		adminSeenIPs[username] = seen
	}
	for oldIP, at := range seen {
		if now.Sub(at) > adminIPRetention {
			delete(seen, oldIP)
		}
	}
	_, known := seen[ip]
	firstObservation := !adminIPBaselined[username]
	seen[ip] = now
	adminIPBaselined[username] = true
	riskMu.Unlock()
	if known || firstObservation {
		return
	}
	sendRiskAlert("admin_new_ip",
		"BotsRouter 安全告警：管理员账号从新 IP 登录",
		alertBody(
			fmt.Sprintf("管理员账号 %s 刚刚从过去 30 天未见过的 IP %s 成功登录。", username, ip),
			fmt.Sprintf("时间：%s", now.Format("2006-01-02 15:04:05")),
			"如非本人操作，请立即修改密码并检查控制台。",
		))
}

// RecordRegistrationBlock notes one registration-guard refusal and
// alerts once an hour when refusals pile up.
func RecordRegistrationBlock(ip string) {
	now := time.Now()
	riskMu.Lock()
	cutoff := now.Add(-regBlockWindow)
	kept := regBlockEvents[:0]
	for _, at := range regBlockEvents {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	regBlockEvents = append(kept, now)
	count := len(regBlockEvents)
	riskMu.Unlock()
	if count < regBlockThreshold {
		return
	}
	sendRiskAlert("registration_block",
		"BotsRouter 安全告警：注册拦截频繁触发",
		alertBody(
			fmt.Sprintf("过去一小时内，注册防重复规则已拦截 %d 次（最近一次来源 IP：%s）。", count, ip),
			fmt.Sprintf("时间：%s", now.Format("2006-01-02 15:04:05")),
			"可能有人在尝试批量注册，请留意。",
		))
}

// RecordTopUpFailure notes one failed redemption attempt and alerts
// when the same user+IP fails repeatedly inside the window.
func RecordTopUpFailure(userId int, ip string) {
	now := time.Now()
	key := fmt.Sprintf("%d|%s", userId, ip)
	riskMu.Lock()
	cutoff := now.Add(-topupFailWindow)
	kept := topupFailEvents[key][:0]
	for _, at := range topupFailEvents[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	topupFailEvents[key] = append(kept, now)
	count := len(topupFailEvents[key])
	riskMu.Unlock()
	if count < topupFailLimit {
		return
	}
	sendRiskAlert("topup_failure",
		"BotsRouter 安全告警：兑换码连续失败",
		alertBody(
			fmt.Sprintf("用户 ID %d（来源 IP：%s）在 10 分钟内兑换失败已达 %d 次。", userId, ip, count),
			fmt.Sprintf("时间：%s", now.Format("2006-01-02 15:04:05")),
			"可能有人在猜测兑换码，请留意。",
		))
}
