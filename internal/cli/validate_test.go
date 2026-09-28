package cli

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/upsidr/merge-gatekeeper/internal/validators"
	"github.com/upsidr/merge-gatekeeper/internal/validators/mock"
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

func Test_waitInitialDelay(t *testing.T) {
	tests := map[string]struct {
		ctx        func() context.Context
		delay      time.Duration
		wantErr    error
		minElapsed time.Duration
		maxElapsed time.Duration
	}{
		"returns immediately when delay is zero": {
			ctx:        context.Background,
			delay:      0,
			wantErr:    nil,
			minElapsed: 0,
			maxElapsed: 100 * time.Millisecond,
		},
		"waits for the given delay": {
			ctx:        context.Background,
			delay:      200 * time.Millisecond,
			wantErr:    nil,
			minElapsed: 200 * time.Millisecond,
			maxElapsed: time.Second,
		},
		"returns ctx error when cancelled during delay": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					time.Sleep(50 * time.Millisecond)
					cancel()
				}()
				return ctx
			},
			delay:      10 * time.Second,
			wantErr:    context.Canceled,
			minElapsed: 50 * time.Millisecond,
			maxElapsed: time.Second,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			err := waitInitialDelay(tt.ctx(), &cobra.Command{}, tt.delay)
			elapsed := time.Since(start)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("waitInitialDelay() error = %v, wantErr %v", err, tt.wantErr)
			}
			if elapsed < tt.minElapsed || elapsed > tt.maxElapsed {
				t.Errorf("waitInitialDelay() took %v, want between %v and %v", elapsed, tt.minElapsed, tt.maxElapsed)
			}
		})
	}
}

func Test_doValidateCmd_initialDelay(t *testing.T) {
	origDelay := initialDelaySecond
	t.Cleanup(func() { initialDelaySecond = origDelay })
	// timeoutSecond is 2 (see TestMain), which is shorter than this delay. If the
	// delay were (wrongly) counted towards the timeout, the context would already
	// be expired when the first validation runs.
	initialDelaySecond = 3

	tests := map[string]struct {
		ctx            func() context.Context
		succeedOnCall  int // 0 means never succeed
		wantErr        error
		wantCalled     bool
		wantMinElapsed time.Duration
	}{
		"delay is not counted towards timeout": {
			ctx:            context.Background,
			succeedOnCall:  2, // succeeds on the second tick, roughly 1 second after the delay
			wantErr:        nil,
			wantCalled:     true,
			wantMinElapsed: time.Duration(initialDelaySecond) * time.Second,
		},
		"returns ctx error without validating when cancelled during delay": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			succeedOnCall:  0,
			wantErr:        context.Canceled,
			wantCalled:     false,
			wantMinElapsed: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			calls := 0
			v := &mock.Validator{
				NameFunc: func() string { return "validator-1" },
				ValidateFunc: func(ctx context.Context) (validators.Status, error) {
					calls++
					ok := tt.succeedOnCall > 0 && calls >= tt.succeedOnCall
					return &mock.Status{
						DetailFunc:    func() string { return "status" },
						IsSuccessFunc: func() bool { return ok },
					}, nil
				},
			}

			start := time.Now()
			err := doValidateCmd(tt.ctx(), &cobra.Command{}, v)
			elapsed := time.Since(start)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("doValidateCmd() error = %v, wantErr %v", err, tt.wantErr)
			}
			if called := calls > 0; called != tt.wantCalled {
				t.Errorf("validator called = %v, want %v", called, tt.wantCalled)
			}
			if elapsed < tt.wantMinElapsed {
				t.Errorf("doValidateCmd() returned after %v, want at least %v", elapsed, tt.wantMinElapsed)
			}
		})
	}
}
