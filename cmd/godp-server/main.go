package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"godp/internal/api/router"
	"godp/internal/config"
)

const banner = `
  ██████   ██████  ██████  ██████  
 ██       ██    ██ ██   ██ ██   ██ 
 ██   ███ ██    ██ ██   ██ ██████  
 ██    ██ ██    ██ ██   ██ ██      
  ██████   ██████  ██████  ██      
  DigitalPersona Biometric Engine for Linux
`

func main() {
	// 1. Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	fmt.Print(banner)

	// 2. Load Configuration
	cfg := config.Load()
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// 3. Setup HTTP Handler / Router
	handler := router.SetupRouter(cfg)

	// 4. Configure HTTP Server
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Run Server in Background
	go func() {
		slog.Info("Starting godp biometric server",
			"address", addr,
			"port", cfg.Port,
			"max_request_mb", cfg.MaxRequestSizeMB,
			"auth_enabled", cfg.APIKey != "",
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed to listen", "error", err)
			os.Exit(1)
		}
	}()

	// 6. Graceful Shutdown on OS Signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	sig := <-stop
	slog.Info("Shutdown signal received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	} else {
		slog.Info("godp server stopped gracefully")
	}
}
