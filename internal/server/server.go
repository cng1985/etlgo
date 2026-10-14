package server

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"etlgo/internal/config"
	"etlgo/internal/handler"
	"etlgo/internal/protocol"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

func New(lc fx.Lifecycle, cfg *config.Config, h *handler.Handler) *http.Server {
	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), cors())
	h.Register(r)
	mountFrontend(r)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					panic(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func mountFrontend(r *gin.Engine) {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		panic(err)
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			protocol.Fail(c, protocol.CodeNotFound, "接口不存在")
			return
		}
		rel := strings.TrimPrefix(path.Clean("/"+c.Request.URL.Path), "/")
		if rel != "" && rel != "." {
			if info, err := fs.Stat(dist, rel); err == nil && !info.IsDir() {
				serveEmbedded(c, dist, rel)
				return
			}
		}
		serveEmbedded(c, dist, "index.html")
	})
}

func serveEmbedded(c *gin.Context, fsys fs.FS, name string) {
	f, err := fsys.Open(name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}
	http.ServeContent(c.Writer, c.Request, name, stat.ModTime(), rs)
}
