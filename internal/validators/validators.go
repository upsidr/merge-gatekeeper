package validators

import (
	"context"
	"errors"
)

var (
	ErrValidate = errors.New("validate error")
)

type Status interface {
	Detail() string
	IsSuccess() bool
}

type Validator interface {
	Name() string
	Validate(ctx context.Context) (Status, error)
}
