package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// MountWeb 托管前端构建产物，并对 Vue history 路由回退到 index.html。
func MountWeb(r *gin.Engine, webDir string) {
	abs, err := filepath.Abs(webDir)
	if err != nil {
		return
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return
	}
	index := filepath.Join(abs, "index.html")

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if p == "/api" || strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		rel := strings.TrimPrefix(path.Clean("/"+p), "/")
		full := filepath.Join(abs, filepath.FromSlash(rel))
		if !isUnder(abs, full) {
			c.Status(http.StatusForbidden)
			return
		}
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			c.File(full)
			return
		}
		c.File(index)
	})
}

func isUnder(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
