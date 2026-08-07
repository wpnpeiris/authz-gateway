package cerbos

import (
	"context"
	"fmt"
	"time"

	cerbossdk "github.com/cerbos/cerbos-sdk-go/cerbos"
	"github.com/wpnpeiris/authz-gateway/internal/authorization"
)

type Authorizer struct {
	client *cerbossdk.GRPCClient
}

func New(address string) (*Authorizer, error) {
	client, err := cerbossdk.New(
		address,
		cerbossdk.WithPlaintext(),
		cerbossdk.WithConnectTimeout(3*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("create cerbos client: %w", err)
	}

	return &Authorizer{
		client: client,
	}, nil
}

func (a *Authorizer) Authorize(ctx context.Context, req authorization.Request) (authorization.Decision, error) {

	principal := cerbossdk.NewPrincipal(
		req.Principal.ID,
		req.Principal.Roles...,
	)

	resource := cerbossdk.NewResource(
		req.Resource.Kind,
		req.Resource.ID,
	)

	batch := cerbossdk.NewResourceBatch()
	batch.Add(resource, req.Action)

	result, err := a.client.CheckResources(
		ctx,
		principal,
		batch,
	)
	if err != nil {
		return authorization.Decision{}, fmt.Errorf(
			"cerbos authorization check: %w",
			err,
		)
	}

	resourceResult := result.GetResource(
		req.Resource.ID,
		cerbossdk.MatchResourceKind(req.Resource.Kind),
	)

	if err := resourceResult.Err(); err != nil {
		return authorization.Decision{}, fmt.Errorf(
			"get cerbos resource result: %w",
			err,
		)
	}

	return authorization.Decision{
		Allowed: resourceResult.IsAllowed(req.Action),
	}, nil
}
