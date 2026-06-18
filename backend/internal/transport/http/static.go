package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func MountStatic(r *gin.Engine, dir string) {
	assetsDir := filepath.Join(dir, "assets")
	if info, err := os.Stat(assetsDir); err == nil && info.IsDir() {
		r.Static("/assets", assetsDir)
	}

	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(dir, "index.html"))
	})

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		relativePath := strings.TrimPrefix(filepath.Clean(c.Request.URL.Path), string(filepath.Separator))
		candidate := filepath.Join(dir, relativePath)

		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			c.File(candidate)
			return
		}

		c.File(filepath.Join(dir, "index.html"))
	})
}
