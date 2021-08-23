package mock

import (
	"context"

	"github.com/upsidr/merge-gatekeeper/internal/validators"
)

type Status struct {
	DetailFunc    func() string
	IsSuccessFunc func() bool
}

func (s *Status) Detail() string {
	return s.DetailFunc()
}

func (s *Status) IsSuccess() bool {
	return s.IsSuccessFunc()
}

type Validator struct {
	NameFunc     func() string
	ValidateFunc func(ctx context.Context) (validators.Status, error)
}

func (v *Validator) Name() string {
	return v.NameFunc()
}

func (v *Validator) Validate(ctx context.Context) (validators.Status, error) {
	return v.ValidateFunc(ctx)
}

var (
	_ validators.Validator = &Validator{}
	_ validators.Status    = &Status{}
)
