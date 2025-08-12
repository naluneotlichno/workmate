package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	backapi "workmate/internal/back/api"
	"workmate/internal/back/config"
	fileutil "workmate/internal/back/file"
	"workmate/internal/back/task"
	frontui "workmate/internal/front/ui"
)

func main() {

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	router := setupRouter()

	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if cfg.DataDir == "data" {
		cfg.DataDir = "storage/data"
	}

	if err := fileutil.EnsureDir(cfg.DataDir); err != nil {
		log.Fatal().Err(err).Str("dir", cfg.DataDir).Msg("ensure data dir")
	}

	taskManager := buildTaskManager(cfg)
	wireAPI(router, taskManager)

	baseCtx, baseCancel := context.WithCancel(context.Background())
	taskManager.SetBaseContext(baseCtx)

	const (
		readHeaderTimeout = 5 * time.Second
		shutdownTimeout   = 10 * time.Second
	)

	srv := newHTTPServer(cfg.Port, router, readHeaderTimeout)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server failed")
		}
	}()

	waitForShutdownSignal()

	gracefulShutdown(srv, baseCancel, taskManager, shutdownTimeout)
}

func setupRouter() *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(backapi.ZerologLogger())
	return r
}

func buildTaskManager(cfg config.Config) *task.Manager {
	tm := task.NewManagerWithOptions(task.Options{
		DataDir:            cfg.DataDir,
		AllowedExtensions:  cfg.AllowedExtensions,
		MaxConcurrentTasks: cfg.MaxConcurrentTasks,
	})

	_ = tm.LoadFromDisk()
	return tm
}

func wireAPI(router *gin.Engine, tm *task.Manager) {
	apiHandler := backapi.NewAPI(tm)
	apiHandler.RegisterRoutes(router)

	uiHandler := frontui.NewUI(tm)
	uiHandler.RegisterRoutes(router)
}

func newHTTPServer(port int, handler http.Handler, readHeaderTimeout time.Duration) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}

func waitForShutdownSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutdown signal received")
}

func gracefulShutdown(srv *http.Server, cancelBase context.CancelFunc, tm *task.Manager, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Warn().Err(err).Msg("http server shutdown warning")
	}

	cancelBase()
	done := tm.WaitAll(ctx)
	if !done {
		log.Warn().Msg("background workers did not finish before timeout")
	}
	log.Info().Msg("server exited cleanly")
}
