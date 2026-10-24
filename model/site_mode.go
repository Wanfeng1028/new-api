package model

import (
	"slices"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

// SiteModeOptionKey is the admin-managed option that records which release
// stage the whole site is in (BotsRouter fork). The official website reads
// it through the public /api/site-mode endpoint and adapts what it shows:
//
//   - beta: payments are not live, displayed prices are test data, the
//     billing formula is not final, and the model library shows the full
//     catalog regardless of channel coverage.
//   - beta_plus: personal top-up and redemption codes are live, prices and
//     the billing formula are final, the audience is a small invited
//     group, and the model library only shows models covered by enabled
//     channels.
//   - release: payment gateways (online Alipay top-up and international
//     payments) are live as well; the site is commercially available.
const SiteModeOptionKey = "SiteMode"

const (
	SiteModeBeta     = "beta"
	SiteModeBetaPlus = "beta_plus"
	SiteModeRelease  = "release"
)

// NormalizeSiteMode maps any stored value onto a known site mode, falling
// back to beta for empty or unrecognized values.
func NormalizeSiteMode(value string) string {
	switch value {
	case SiteModeBetaPlus, SiteModeRelease:
		return value
	default:
		return SiteModeBeta
	}
}

var siteModeUnsetWarnOnce sync.Once

// GetSiteMode returns the current site mode.
func GetSiteMode() string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	raw := common.OptionMap[SiteModeOptionKey]
	if raw == "" {
		// BotsRouter fork patch (review F-16): an unset option silently
		// falls back to beta (payments closed). Warn once so a lost
		// option row is visible in logs instead of silently changing
		// payment behaviour.
		siteModeUnsetWarnOnce.Do(func() {
			common.SysLog("site mode option is unset, falling back to beta (payments closed)")
		})
	}
	return NormalizeSiteMode(raw)
}

// GetConnectedModels returns the sorted names of the models covered by at
// least one enabled channel (an enabled ability row on an enabled
// channel). The official website uses it to filter its model library in
// beta_plus and release modes.
func GetConnectedModels() []string {
	models := make([]string, 0)
	DB.Table("abilities").
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities.enabled = ? AND channels.status = ?", true, common.ChannelStatusEnabled).
		Distinct("abilities.model").
		Pluck("abilities.model", &models)
	slices.Sort(models)
	return models
}
