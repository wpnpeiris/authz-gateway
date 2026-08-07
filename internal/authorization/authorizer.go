package authorization

import "context"

type Authorizer interface {
	Authorize(ctx context.Context, req Request) (Decision, error)
}
