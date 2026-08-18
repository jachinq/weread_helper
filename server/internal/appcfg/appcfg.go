package appcfg

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jachin/weread-helper/internal/config"
	"github.com/jachin/weread-helper/internal/secret"
	"github.com/jachin/weread-helper/internal/store"
)

const (
	DefaultSkillVersion = "1.0.4"
	DefaultGatewayURL   = "https://i.weread.qq.com/api/agent/gateway"
	DefaultSyncInterval = "6h"
	DefaultSiteTitle    = "纸间笔记"
	DefaultTheme        = "walnut"
	DefaultColorScheme  = "dark"
)

var allowedThemes = map[string]struct{}{
	"walnut":      {},
	"celadon":     {},
	"cinnabar":    {},
	"inknight":    {},
	"moss":        {},
	"moonfoil":    {},
	"persimmon":   {},
	"letterpress": {},
	"begonia":     {},
}

func NormalizeTheme(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if _, ok := allowedThemes[s]; ok {
		return s
	}
	return DefaultTheme
}

func NormalizeColorScheme(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "light" || s == "dark" {
		return s
	}
	return DefaultColorScheme
}

type Runtime struct {
	APIKey       string
	SkillVersion string
	GatewayURL   string
	SyncInterval time.Duration
	SiteTitle    string
}

func ParseInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		s = DefaultSyncInterval
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("无效的同步间隔")
	}
	return d, nil
}

func ValidateGateway(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("Gateway URL 须为 http(s) 地址")
	}
	return nil
}

func LoadRuntime(st *store.Store, encKey []byte, env config.Config) (Runtime, error) {
	row, err := st.LoadAppSettings()
	if err != nil {
		return Runtime{}, err
	}

	rt := Runtime{
		SkillVersion: or(row.SkillVersion, env.SkillVersion, DefaultSkillVersion),
		GatewayURL:   or(row.GatewayURL, env.GatewayURL, DefaultGatewayURL),
		SiteTitle:    or(row.SiteTitle, DefaultSiteTitle),
	}

	intervalSrc := or(row.SyncInterval, env.SyncInterval.String(), DefaultSyncInterval)
	d, err := ParseInterval(intervalSrc)
	if err != nil {
		d = 6 * time.Hour
	}
	rt.SyncInterval = d

	if row.APIKeyCipher != "" {
		plain, err := secret.Decrypt(encKey, row.APIKeyCipher)
		if err != nil {
			return Runtime{}, fmt.Errorf("解密 API Key 失败: %w", err)
		}
		rt.APIKey = plain
	} else if strings.TrimSpace(env.APIKey) != "" {
		rt.APIKey = strings.TrimSpace(env.APIKey)
		if err := persistAPIKey(st, encKey, rt.APIKey); err != nil {
			return Runtime{}, err
		}
	}

	if row.SkillVersion == "" {
		_ = st.SetSetting("skill_version", rt.SkillVersion)
	}
	if row.GatewayURL == "" {
		_ = st.SetSetting("gateway_url", rt.GatewayURL)
	}
	if row.SyncInterval == "" {
		src := DefaultSyncInterval
		if env.SyncInterval > 0 {
			src = env.SyncInterval.String()
		}
		_ = st.SetSetting("sync_interval", src)
	}
	if row.SiteTitle == "" {
		_ = st.SetSetting("site_title", rt.SiteTitle)
	}
	if row.Theme == "" {
		_ = st.SetSetting("theme", DefaultTheme)
	}
	if row.ColorScheme == "" {
		_ = st.SetSetting("color_scheme", DefaultColorScheme)
	}

	return rt, nil
}

func persistAPIKey(st *store.Store, encKey []byte, plain string) error {
	cipher, err := secret.Encrypt(encKey, plain)
	if err != nil {
		return err
	}
	return st.SetSetting("api_key", cipher)
}

func Save(st *store.Store, encKey []byte, skill, gateway, interval, title, theme, scheme, newAPIKey string, keepKey bool, currentKey string) (Runtime, error) {
	skill = strings.TrimSpace(skill)
	gateway = strings.TrimSpace(gateway)
	interval = strings.TrimSpace(interval)
	title = strings.TrimSpace(title)
	newAPIKey = strings.TrimSpace(newAPIKey)

	if skill == "" {
		skill = DefaultSkillVersion
	}
	if gateway == "" {
		gateway = DefaultGatewayURL
	}
	if title == "" {
		title = DefaultSiteTitle
	}
	theme = NormalizeTheme(theme)
	scheme = NormalizeColorScheme(scheme)
	if err := ValidateGateway(gateway); err != nil {
		return Runtime{}, err
	}
	d, err := ParseInterval(interval)
	if err != nil {
		return Runtime{}, err
	}

	apiKey := currentKey
	if !keepKey && newAPIKey != "" {
		apiKey = newAPIKey
		if err := persistAPIKey(st, encKey, apiKey); err != nil {
			return Runtime{}, err
		}
	}

	if err := st.SetSetting("skill_version", skill); err != nil {
		return Runtime{}, err
	}
	if err := st.SetSetting("gateway_url", gateway); err != nil {
		return Runtime{}, err
	}
	if err := st.SetSetting("sync_interval", interval); err != nil {
		return Runtime{}, err
	}
	if err := st.SetSetting("site_title", title); err != nil {
		return Runtime{}, err
	}
	if err := st.SetSetting("theme", theme); err != nil {
		return Runtime{}, err
	}
	if err := st.SetSetting("color_scheme", scheme); err != nil {
		return Runtime{}, err
	}

	return Runtime{
		APIKey:       apiKey,
		SkillVersion: skill,
		GatewayURL:   gateway,
		SyncInterval: d,
		SiteTitle:    title,
	}, nil
}

func or(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
