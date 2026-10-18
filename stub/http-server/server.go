package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/morya/spgw/internal/router/middle"
	routerhelper "github.com/morya/spgw/internal/router/router_helper"
)

type Server struct {
	httpSrv *http.Server
	engine  *gin.Engine
	cfg     *Config
}

func NewServer(cfg *Config, app *App) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	engine.Use(gin.Recovery(), middle.BodyDump(), middle.BodyLimit(4096<<10))

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	api := engine.Group("/api/sms", app.signAuthMiddle())

	api.POST("/send", routerhelper.HandleWithModel(app.handleSendOne))
	api.POST("/batchSend/v2", routerhelper.HandleWithModel(app.handleBatchSend))

	return &Server{
		httpSrv: &http.Server{Addr: cfg.Listen, Handler: engine},
		engine:  engine,
		cfg:     cfg,
	}
}

func (s *Server) ListenAndServe() error {
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutCtx)
		return nil
	case err := <-errCh:
		return err
	case <-time.After(0):
		return nil
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}
