package validators

import (
	"context"
)

type Status interface {
	Detail() string
	IsSuccess() bool
}

type Validator interface {
	Name() string
	Validate(ctx context.Context) (Status, error)
}
