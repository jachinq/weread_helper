package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jachin/weread-helper/internal/appcfg"
	"github.com/jachin/weread-helper/internal/secret"
)

type settingsDTO struct {
	APIKeyMasked string `json:"apiKeyMasked"`
	SkillVersion string `json:"skillVersion"`
	GatewayURL   string `json:"gatewayUrl"`
	SyncInterval string `json:"syncInterval"`
	SiteTitle    string `json:"siteTitle"`
}

func (s *Server) settingsPublic() (settingsDTO, error) {
	row, err := s.store.LoadAppSettings()
	if err != nil {
		return settingsDTO{}, err
	}
	plain, err := secret.Decrypt(s.encKey, row.APIKeyCipher)
	if err != nil {
		return settingsDTO{}, err
	}
	skill := strings.TrimSpace(row.SkillVersion)
	if skill == "" {
		skill = appcfg.DefaultSkillVersion
	}
	gw := strings.TrimSpace(row.GatewayURL)
	if gw == "" {
		gw = appcfg.DefaultGatewayURL
	}
	interval := strings.TrimSpace(row.SyncInterval)
	if interval == "" {
		interval = appcfg.DefaultSyncInterval
	}
	title := strings.TrimSpace(row.SiteTitle)
	if title == "" {
		title = appcfg.DefaultSiteTitle
	}
	return settingsDTO{
		APIKeyMasked: secret.MaskAPIKey(plain),
		SkillVersion: skill,
		GatewayURL:   gw,
		SyncInterval: interval,
		SiteTitle:    title,
	}, nil
}

func (s *Server) settingsGet(c *gin.Context) {
	dto, err := s.settingsPublic()
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}

type settingsPutBody struct {
	APIKey       string `json:"apiKey"`
	SkillVersion string `json:"skillVersion"`
	GatewayURL   string `json:"gatewayUrl"`
	SyncInterval string `json:"syncInterval"`
	SiteTitle    string `json:"siteTitle"`
}

func (s *Server) settingsPut(c *gin.Context) {
	var body settingsPutBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
		return
	}

	row, err := s.store.LoadAppSettings()
	if err != nil {
		writeErr(c, err)
		return
	}
	current, err := secret.Decrypt(s.encKey, row.APIKeyCipher)
	if err != nil {
		writeErr(c, err)
		return
	}

	keep := strings.TrimSpace(body.APIKey) == ""
	rt, err := appcfg.Save(
		s.store,
		s.encKey,
		body.SkillVersion,
		body.GatewayURL,
		body.SyncInterval,
		body.SiteTitle,
		body.APIKey,
		keep,
		current,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.client.Update(rt.GatewayURL, rt.APIKey, rt.SkillVersion)
	s.job.ApplyRuntime(rt.SyncInterval)

	dto, err := s.settingsPublic()
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, dto)
}
