package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"

	"github.com/wpnpeiris/authz-gateway/internal/authorization"
	"github.com/wpnpeiris/authz-gateway/internal/authorization/cerbos"
	"github.com/wpnpeiris/authz-gateway/internal/logging"
	"github.com/wpnpeiris/authz-gateway/internal/metrics"
)

type GatewayServer struct {
	logger     log.Logger
	config     Config
	authorizer authorization.Authorizer
}

func (s *GatewayServer) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type Config struct {
	Endpoint          string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// NewGatewayServer creates a GatewayServer from the provided options.
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

	authorizer, err := cerbos.New(opts.CerbosAddress)
	if err != nil {
		return nil, fmt.Errorf("initialize cerbos authorizer: %w", err)
	}
	return &GatewayServer{
		logger:     logger,
		config:     config,
		authorizer: authorizer,
	}, nil
}

// Start starts the HTTP server with the provided configuration and blocks until it exits.
func (s *GatewayServer) Start() error {
	logging.Info(s.logger, "msg", "Starting authz gateway server...")
	router := mux.NewRouter().SkipClean(true)

	metrics.RegisterMetricEndpoint(router)
	router.Methods(http.MethodGet).Path("/healthz").HandlerFunc(s.Healthz)
	router.Methods(http.MethodGet).Path("/authorize").HandlerFunc(s.Authorize)

	srv := &http.Server{
		Addr:              s.config.Endpoint,
		Handler:           router,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		MaxHeaderBytes:    128 << 10, // 128 KiB
	}

	// Channel to listen for errors from the HTTP server
	serverErrors := make(chan error, 1)

	// Start HTTP server in a goroutine
	go func() {
		logging.Info(s.logger, "msg", "Listening for HTTP requests", "address", s.config.Endpoint)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdown)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logging.Info(s.logger, "msg", "Received shutdown signal", "signal", sig)

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
