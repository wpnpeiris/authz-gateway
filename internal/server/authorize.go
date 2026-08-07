package server

import (
	"github.com/wpnpeiris/authz-gateway/internal/authorization"
	"github.com/wpnpeiris/authz-gateway/internal/logging"
	"net/http"
)

func (s *GatewayServer) Authorize(w http.ResponseWriter, r *http.Request) {
	req, err := authorization.RequestFromHeaders(r.Header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
			"principal", req.Principal.ID,
			"resource_kind", req.Resource.Kind,
			"resource_id", req.Resource.ID,
			"action", req.Action,
		)

		w.WriteHeader(http.StatusForbidden)
		return
	}

	logging.Info(
		s.logger,
		"msg", "Authorization allowed",
		"principal", req.Principal.ID,
		"resource_kind", req.Resource.Kind,
		"resource_id", req.Resource.ID,
		"action", req.Action,
	)

	w.WriteHeader(http.StatusOK)
}
