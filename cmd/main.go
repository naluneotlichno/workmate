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

	"workmate/internal/api"
	"workmate/internal/config"
	"workmate/internal/task"
)

func main() {
	// Setup zerolog
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	router := setupRouter()

	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	taskManager := buildTaskManager(cfg)
	wireAPI(router, taskManager)

	// Prepare a shutdown-aware context for long-running ops (downloads)
	baseCtx, baseCancel := context.WithCancel(context.Background())
	taskManager.SetBaseContext(baseCtx)

	// Graceful HTTP server
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

	// Stop accepting new connections and finish in-flight requests
	gracefulShutdown(srv, baseCancel, taskManager, shutdownTimeout)
}

func setupRouter() *gin.Engine {
	r := gin.New()
	// Build API to register UI template before routes (Gin warns otherwise)
	// UI template is set during UI route registration
	r.Use(gin.Recovery())
	r.Use(api.ZerologLogger())
	return r
}

func buildTaskManager(cfg config.Config) *task.Manager {
	tm := task.NewManagerWithOptions(task.Options{
		DataDir:            cfg.DataDir,
		AllowedExtensions:  cfg.AllowedExtensions,
		MaxConcurrentTasks: cfg.MaxConcurrentTasks,
	})
	// best-effort reload from disk
	_ = tm.LoadFromDisk()
	return tm
}

func wireAPI(router *gin.Engine, tm *task.Manager) {
	apiHandler := api.NewAPI(tm)
	// Register UI first to call SetHTMLTemplate before other routes
	apiHandler.RegisterUIRoutes(router)
	apiHandler.RegisterRoutes(router)
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
	// Wait for background workers to finish
	cancelBase() // cancel downloads
	done := tm.WaitAll(ctx)
	if !done {
		log.Warn().Msg("background workers did not finish before timeout")
	}
	log.Info().Msg("server exited cleanly")
}
