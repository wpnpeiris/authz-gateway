package server

import (
	"context"
	"fmt"
	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/wpnpeiris/authz-gateway/internal/logging"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type GatewayServer struct {
	logger log.Logger
	config Config
}

type Config struct {
	Endpoint          string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// LogAndExit logs an error message to stderr and exits with status code 1.
func LogAndExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

// NewGatewayServer constructs a server that ...
func NewGatewayServer(opts *Options) (*GatewayServer, error) {
	logger := logging.NewLogger(logging.Config{
		Format: opts.LogFormat,
		Level:  opts.LogLevel,
	})

	config := Config{
		Endpoint:          opts.ServerListen,
		ReadTimeout:       opts.ReadTimeout,
		WriteTimeout:      opts.WriteTimeout,
		IdleTimeout:       opts.IdleTimeout,
		ReadHeaderTimeout: opts.ReadHeaderTimeout,
	}

	return &GatewayServer{logger, config}, nil
}

// Start starts the HTTP server with the provided configuration and blocks until it exits.
func (s *GatewayServer) Start() error {
	logging.Info(s.logger, "msg", fmt.Sprintf("Starting authz gateway server..."))
	router := mux.NewRouter().SkipClean(true)

	srv := &http.Server{
		Addr:              s.config.Endpoint,
		Handler:           router,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Channel to listen for errors from the HTTP server
	serverErrors := make(chan error, 1)

	// Start HTTP server in a goroutine
	go func() {
		logging.Info(s.logger, "msg", fmt.Sprintf("Listening for HTTP requests on %s", s.config.Endpoint))
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logging.Info(s.logger, "msg", fmt.Sprintf("Received signal %v, starting graceful shutdown", sig))

		// Give outstanding requests 30 seconds to complete
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logging.Error(s.logger, "msg", "Error during server shutdown", "err", err)
			return fmt.Errorf("could not gracefully shutdown server: %w", err)
		}

	}

	return nil

}
