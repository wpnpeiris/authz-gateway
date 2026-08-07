package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/cerbos/cerbos-sdk-go/cerbos"
	"github.com/wpnpeiris/authz-gateway/internal/authorization"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"

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

func (s *GatewayServer) Authorize(w http.ResponseWriter, r *http.Request) {
	principalID := r.Header.Get("X-Authz-Principal")
	if principalID == "" {
		http.Error(w, "missing principal", http.StatusUnauthorized)
		return
	}

	method := r.Header.Get("X-Forwarded-Method")
	path := r.Header.Get("X-Forwarded-Uri")

	if method == "" || path == "" {
		http.Error(
			w,
			"missing forwarded request information",
			http.StatusBadRequest,
		)
		return
	}

	resource, action, ok := mapRequest(method, path)
	if !ok {
		http.Error(w, "request is not authorized", http.StatusForbidden)
		return
	}

	decision, err := s.authorizer.Authorize(
		r.Context(),
		authorization.Request{
			Principal: authorization.Principal{
				ID:    principalID,
				Roles: []string{"customer_application"},
			},
			Resource: resource,
			Action:   action,
		},
	)

	if err != nil {
		logging.Error(
			s.logger,
			"msg", "Authorization provider failed",
			"err", err,
		)

		http.Error(
			w,
			"authorization service unavailable",
			http.StatusServiceUnavailable,
		)
		return
	}

	if !decision.Allowed {
		logging.Info(
			s.logger,
			"msg", "Authorization denied",
			"principal", principalID,
			"resource_kind", resource.Kind,
			"resource_id", resource.ID,
			"action", action,
		)

		w.WriteHeader(http.StatusForbidden)
		return
	}

	logging.Info(
		s.logger,
		"msg", "Authorization allowed",
		"principal", principalID,
		"resource_kind", resource.Kind,
		"resource_id", resource.ID,
		"action", action,
	)

	w.WriteHeader(http.StatusOK)
}

func mapRequest(
	method string,
	path string,
) (authorization.Resource, string, bool) {

	const prefix = "/api/v1/anything/"

	if !strings.HasPrefix(path, prefix) {
		return authorization.Resource{}, "", false
	}

	resourceID := strings.TrimPrefix(path, prefix)
	if resourceID == "" || strings.Contains(resourceID, "/") {
		return authorization.Resource{}, "", false
	}

	var action string

	switch method {
	case http.MethodGet:
		action = "read"
	case http.MethodPut, http.MethodPatch:
		action = "update"
	case http.MethodDelete:
		action = "delete"
	default:
		return authorization.Resource{}, "", false
	}

	return authorization.Resource{
		Kind: "anything",
		ID:   resourceID,
	}, action, true
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

	authorizer, _ := authorization.New(opts.CerbosAddress)
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
