package validators

import (
	"context"
	"errors"
)

var (
	ErrValidate = errors.New("validate error")
)

type Validator interface {
	Validate(ctx context.Context) error
}
