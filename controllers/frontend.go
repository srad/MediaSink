package controllers

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func serveFrontend(router *gin.Engine, frontendFS embed.FS, version, commit, apiVersion string) {
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic("failed to sub frontend/dist from embedded FS: " + err.Error())
	}

	// env.js: runtime config derived from the serving host for same-origin deployment.
	// Using window.location for WebSocket protocol so it works with both http and https.
	router.GET("/env.js", func(c *gin.Context) {
		wsProto := "ws"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			wsProto = "wss"
		}
		c.Header("Content-Type", "application/javascript; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		fmt.Fprintf(c.Writer, `window.APP_APIURL = "/api/v1";
window.APP_BASE = "";
window.APP_NAME = "MediaSink";
window.APP_SOCKETURL = "%s://%s/api/v1/ws";
window.APP_FILEURL = "/videos";
`, wsProto, c.Request.Host)
	})

	// build.js: build metadata injected from Go ldflags at compile time.
	router.GET("/build.js", func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		fmt.Fprintf(c.Writer, `window.APP_BUILD = "%s";
window.APP_VERSION = "%s";
window.APP_API_VERSION = "%s";
`, commit, version, apiVersion)
	})

	// SPA: serve embedded static assets with index.html fallback for client-side routes.
	fileServer := http.FileServer(http.FS(distFS))
	router.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		if f, err := distFS.Open(path); err == nil {
			f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// SPA fallback: any unmatched path serves index.html for client-side routing.
		indexContent, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.String(http.StatusServiceUnavailable, "Frontend not built. Run: cd frontend && pnpm install && pnpm build")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexContent)
	})
}
