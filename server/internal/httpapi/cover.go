package httpapi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const coverMaxBytes = 8 << 20

var coverClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return validateCoverURL(req.URL)
	},
}

func (s *Server) coverProxy(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("url"))
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing url"})
		return
	}
	u, err := url.Parse(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}
	u = unwrapNestedCoverProxy(u)
	if u.Scheme == "http" {
		u.Scheme = "https"
	}
	if err := validateCoverURL(u); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://weread.qq.com/")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")

	resp, err := coverClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "fetch cover failed"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.Status(http.StatusBadGateway)
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, coverMaxBytes))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "fetch cover failed"})
		return
	}
	ct := resp.Header.Get("Content-Type")
	if sniffed := sniffImageType(body); sniffed != "" {
		ct = sniffed
	} else if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "image/") {
		ct = "image/jpeg"
	}
	c.Header("Content-Type", ct)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, ct, body)
}

func unwrapNestedCoverProxy(u *url.URL) *url.URL {
	if u == nil {
		return u
	}
	host := strings.ToLower(u.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return u
	}
	if u.Path != "/api/covers" && !strings.HasSuffix(u.Path, "/api/covers") {
		return u
	}
	inner := strings.TrimSpace(u.Query().Get("url"))
	if inner == "" {
		return u
	}
	parsed, err := url.Parse(inner)
	if err != nil {
		return u
	}
	return parsed
}

func validateCoverURL(u *url.URL) error {
	if u == nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return errors.New("only https cover urls are allowed")
	}
	host := strings.ToLower(u.Hostname())
	if !allowedCoverHost(host) {
		return fmt.Errorf("host not allowed: %s", host)
	}
	if u.Port() != "" && u.Port() != "443" {
		return errors.New("only https port 443 is allowed")
	}
	return nil
}

func allowedCoverHost(host string) bool {
	if host == "weread.qq.com" || host == "qpic.cn" || host == "qqmail.com" {
		return true
	}
	for _, s := range []string{".weread.qq.com", ".myqcloud.com", ".qpic.cn", ".qcloud.com", ".qqmail.com", ".tencent-cloud.com"} {
		if strings.HasSuffix(host, s) {
			return true
		}
	}
	return false
}

func sniffImageType(b []byte) string {
	switch {
	case len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		return "image/jpeg"
	case bytes.HasPrefix(b, []byte("\x89PNG")):
		return "image/png"
	case bytes.HasPrefix(b, []byte("GIF8")):
		return "image/gif"
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "image/webp"
	default:
		return ""
	}
}
