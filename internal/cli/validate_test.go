package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/upsidr/check-other-job-status/internal/validators"
	"github.com/upsidr/check-other-job-status/internal/validators/mock"
)

func Test_doValidateCmd(t *testing.T) {
	type args struct {
		ctx context.Context
		cmd *cobra.Command
		vs  []validators.Validator
	}
	type test struct {
		args    args
		wantErr bool
	}
	tests := map[string]test{
		"returns nil when the validation is success": func() test {
			validateInvalSecond = 1
			timeoutSecond = 2

			return test{
				args: args{
					ctx: context.Background(),
					cmd: &cobra.Command{},
					vs: []validators.Validator{
						&mock.Validator{
							ValidateFunc: func(ctx context.Context) error { return nil },
						},
						&mock.Validator{
							ValidateFunc: func(ctx context.Context) error { return nil },
						},
					},
				},
			}
		}(),
		"returns error when the validation is timeout": func() test {
			validateInvalSecond = 1
			timeoutSecond = 2

			return test{
				args: args{
					ctx: context.Background(),
					cmd: &cobra.Command{},
					vs: []validators.Validator{
						&mock.Validator{
							ValidateFunc: func(ctx context.Context) error { return validators.ErrValidate },
						},
					},
				},
				wantErr: true,
			}
		}(),
		"returns error when the validator returns invalid error": func() test {
			validateInvalSecond = 1
			timeoutSecond = 2

			return test{
				args: args{
					ctx: context.Background(),
					cmd: &cobra.Command{},
					vs: []validators.Validator{
						&mock.Validator{
							ValidateFunc: func(ctx context.Context) error { return errors.New("err") },
						},
					},
				},
				wantErr: true,
			}
		}(),
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := doValidateCmd(tt.args.ctx, tt.args.cmd, tt.args.vs...); (err != nil) != tt.wantErr {
				t.Errorf("doValidateCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
