package cli

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/upsidr/check-other-job-status/internal/validators"
	"github.com/upsidr/check-other-job-status/internal/validators/mock"
)

func TestMain(m *testing.M) {
	validateInvalSecond = 1
	timeoutSecond = 2
	os.Exit(m.Run())
}

func Test_ownerAndRepository(t *testing.T) {
	tests := map[string]struct {
		str       string
		wantOwner string
		wantRepo  string
	}{
		"returns empty when str is empty": {
			str:       "",
			wantOwner: "",
			wantRepo:  "",
		},
		"returns (upsidr, repo) when str is upsidr/repo": {
			str:       "upsidr/repo",
			wantOwner: "upsidr",
			wantRepo:  "repo",
		},
		"returns (upsidr, '') when str is upsidr": {
			str:       "upsidr",
			wantOwner: "upsidr",
			wantRepo:  "",
		},
		"returns ('', repo) when str is /repo": {
			str:       "/repo",
			wantOwner: "",
			wantRepo:  "repo",
		},
		"returns (upsidr, repo/repo) when str is upsidr/repo/repo": {
			str:       "upsidr/repo/repo",
			wantOwner: "upsidr",
			wantRepo:  "repo/repo",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotOwner, gotRepo := ownerAndRepository(tt.str)
			if gotOwner != tt.wantOwner {
				t.Errorf("ownerAndRepository() owner = %s, wantOwner: %s", gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("ownerAndRepository() repo = %s, wantOwner: %s", gotRepo, tt.wantRepo)
			}
		})
	}
}

func Test_doValidateCmd(t *testing.T) {
	tests := map[string]struct {
		ctx     context.Context
		cmd     *cobra.Command
		vs      []validators.Validator
		wantErr bool
	}{
		"returns nil when the validation is success": {
			ctx: context.Background(),
			cmd: &cobra.Command{},
			vs: []validators.Validator{
				&mock.Validator{
					NameFunc: func() string { return "validator-1" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return &mock.Status{
							DetailFunc:    func() string { return "success-1" },
							IsSuccessFunc: func() bool { return true },
						}, nil
					},
				},
				&mock.Validator{
					NameFunc: func() string { return "validator-2" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return &mock.Status{
							DetailFunc:    func() string { return "success-2" },
							IsSuccessFunc: func() bool { return true },
						}, nil
					},
				},
			},
			wantErr: false,
		},
		"returns error when the validation timed out": {
			ctx: context.Background(),
			cmd: &cobra.Command{},
			vs: []validators.Validator{
				&mock.Validator{
					NameFunc: func() string { return "validator-1" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return &mock.Status{
							DetailFunc:    func() string { return "fails-1" },
							IsSuccessFunc: func() bool { return false },
						}, nil
					},
				},
				&mock.Validator{
					NameFunc: func() string { return "validator-2" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return &mock.Status{
							DetailFunc:    func() string { return "fails-2" },
							IsSuccessFunc: func() bool { return false },
						}, nil
					},
				},
			},
			wantErr: true,
		},
		"returns error when the validator return an error": {
			ctx: context.Background(),
			cmd: &cobra.Command{},
			vs: []validators.Validator{
				&mock.Validator{
					NameFunc: func() string { return "validator-1" },
					ValidateFunc: func(ctx context.Context) (validators.Status, error) {
						return nil, errors.New("err")
					},
				},
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := doValidateCmd(tt.ctx, tt.cmd, tt.vs...); (err != nil) != tt.wantErr {
				t.Errorf("doValidateCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
