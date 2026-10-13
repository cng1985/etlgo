package server

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"etlgo/internal/config"
	"etlgo/internal/handler"

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
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func mountFrontend(r *gin.Engine) {
	dist := "frontend/dist"
	info, err := os.Stat(dist)
	if err != nil || !info.IsDir() {
		return
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在"})
			return
		}
		rel := strings.TrimPrefix(filepath.Clean(c.Request.URL.Path), "/")
		file := filepath.Join(dist, rel)
		if rel != "" && rel != "." {
			if st, err := os.Stat(file); err == nil && !st.IsDir() {
				c.File(file)
				return
			}
		}
		c.File(filepath.Join(dist, "index.html"))
	})
}
