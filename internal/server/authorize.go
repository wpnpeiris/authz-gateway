package server

import (
	"net/http"
	"strings"

	"github.com/wpnpeiris/authz-gateway/internal/authorization"
	"github.com/wpnpeiris/authz-gateway/internal/logging"
)

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
				Roles: []string{"some_application"},
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

	const prefix = "/api/v1/some_resource/"

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
