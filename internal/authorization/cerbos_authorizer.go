package authorization

import (
	"context"
	"fmt"
	cerbossdk "github.com/cerbos/cerbos-sdk-go/cerbos"
)

type CerbosAuthorizer struct {
	client *cerbossdk.GRPCClient
}

func New(address string) (*CerbosAuthorizer, error) {
	client, err := cerbossdk.New(address, cerbossdk.WithPlaintext())
	if err != nil {
		return nil, fmt.Errorf("create cerbos client: %w", err)
	}

	return &CerbosAuthorizer{
		client: client,
	}, nil
}

func (a *CerbosAuthorizer) Authorize(ctx context.Context, req Request) (Decision, error) {

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
		return Decision{}, fmt.Errorf(
			"cerbos authorization check: %w",
			err,
		)
	}

	allowed := result.IsAllowed(
		req.Resource.ID,
		req.Action,
	)

	return Decision{
		Allowed: allowed,
	}, nil
}
