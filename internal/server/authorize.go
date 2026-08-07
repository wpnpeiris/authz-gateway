package server

import (
	"net/http"
	"strings"

	"github.com/wpnpeiris/authz-gateway/internal/authorization"
	"github.com/wpnpeiris/authz-gateway/internal/logging"
)

func parseRoles(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	roles := make([]string, 0, len(parts))

	for _, role := range parts {
		role = strings.TrimSpace(role)
		if role != "" {
			roles = append(roles, role)
		}
	}

	return roles
}

func (s *GatewayServer) Authorize(w http.ResponseWriter, r *http.Request) {
	principalID := r.Header.Get("X-Authz-Principal")
	resourceKind := r.Header.Get("X-Authz-Resource")
	resourceID := r.Header.Get("X-Authz-Resource-ID")
	action := r.Header.Get("X-Authz-Action")

	if principalID == "" ||
		resourceKind == "" ||
		resourceID == "" ||
		action == "" {

		http.Error(w, "missing required authorization headers", http.StatusBadRequest)
		return
	}

	roles := parseRoles(r.Header.Get("X-Authz-Roles"))
	req := authorization.Request{
		Principal: authorization.Principal{
			ID:    principalID,
			Roles: roles,
		},
		Resource: authorization.Resource{
			Kind: resourceKind,
			ID:   resourceID,
		},
		Action: action,
	}

	decision, err := s.authorizer.Authorize(r.Context(), req)
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
			"resource_kind", resourceKind,
			"resource_id", resourceID,
			"action", action,
		)

		w.WriteHeader(http.StatusForbidden)
		return
	}

	logging.Info(
		s.logger,
		"msg", "Authorization allowed",
		"principal", principalID,
		"resource_kind", resourceKind,
		"resource_id", resourceID,
		"action", action,
	)

	w.WriteHeader(http.StatusOK)
}
