package status

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/upsidr/check-other-job-status/internal/github"
	"github.com/upsidr/check-other-job-status/internal/github/mock"
	"github.com/upsidr/check-other-job-status/internal/validators"
)

func stringPtr(str string) *string {
	return &str
}

func TestCreateValidator(t *testing.T) {
	type args struct {
		c    github.Client
		opts []Option
	}
	type test struct {
		args    args
		want    validators.Validator
		wantErr bool
	}
	tests := map[string]test{
		"returns Validator when option is empty": func() test {
			c := &mock.Client{}
			return test{
				args: args{
					c: c,
				},
				want:    nil,
				wantErr: true,
			}
		}(),
		"returns Validator when option is not empty": func() test {
			c := &mock.Client{}
			return test{
				args: args{
					c: c,
					opts: []Option{
						WithGitHubOwnerAndRepo("test-owner", "test-repo"),
						WithGitHubRef("sha"),
						WithTargetJob("job"),
					},
				},
				want: &statusValidator{
					client:        c,
					owner:         "test-owner",
					repo:          "test-repo",
					ref:           "sha",
					targetJobName: "job",
				},
				wantErr: false,
			}
		}(),
		"returns Validator when there are duplicate options": func() test {
			c := &mock.Client{}
			return test{
				args: args{
					c: c,
					opts: []Option{
						WithGitHubOwnerAndRepo("test", "test-repo"),
						WithGitHubRef("sha"),
						WithGitHubRef("sha-01"),
						WithTargetJob("job"),
						WithTargetJob("job-01"),
					},
				},
				want: &statusValidator{
					client:        c,
					owner:         "test",
					repo:          "test-repo",
					ref:           "sha-01",
					targetJobName: "job-01",
				},
				wantErr: false,
			}
		}(),
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := CreateValidator(tt.args.c, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateValidator error = %v, wantErr: %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateValidator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_statusValidator_Validate(t *testing.T) {
	type fields struct {
		token         string
		repo          string
		owner         string
		ref           string
		targetJobName string
		client        github.Client
	}
	type test struct {
		fields  fields
		ctx     context.Context
		wantErr bool
	}
	tests := map[string]test{
		"returns nil when validation is success": func() test {
			var (
				testOwner     = "test-owner"
				testRepo      = "test-repo"
				testRef       = "sha"
				targetJobName = "target-job"
			)

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr(targetJobName),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:        c,
					targetJobName: targetJobName,
					owner:         testOwner,
					repo:          testRepo,
					ref:           testRef,
				},
				wantErr: false,
			}
		}(),
		"returns nil when there is one job": func() test {
			var (
				testOwner     = "test-owner"
				testRepo      = "test-repo"
				testRef       = "sha"
				targetJobName = "target-job"
			)

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []github.RepoStatus{
							{
								Context: stringPtr(targetJobName),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:        c,
					targetJobName: targetJobName,
					owner:         testOwner,
					repo:          testRepo,
					ref:           testRef,
				},
				wantErr: false,
			}
		}(),
		"returns error when listStatuses returns an error": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return nil, nil, errors.New("error")
				},
			}
			return test{
				fields: fields{
					client: c,
				},
				wantErr: true,
			}
		}(),
		"returns validate error when the returned job status is empty": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []github.RepoStatus{},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			}
			return test{
				fields: fields{
					client: c,
				},
				wantErr: false,
			}
		}(),
		"returns validate error when the success job count is invalid": func() test {
			targetJobName := "target-job"

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(errorState),
							},
							{
								Context: stringPtr(targetJobName),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:        c,
					targetJobName: targetJobName,
				},
				wantErr: true,
			}
		}(),
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sv := &statusValidator{
				repo:          tt.fields.repo,
				owner:         tt.fields.owner,
				ref:           tt.fields.ref,
				targetJobName: tt.fields.targetJobName,
				client:        tt.fields.client,
			}
			if err := sv.Validate(tt.ctx); (err != nil) != tt.wantErr {
				t.Errorf("statusValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_statusValidator_listStatues(t *testing.T) {
	type fields struct {
		token         string
		repo          string
		owner         string
		ref           string
		targetJobName string
		client        github.Client
	}
	type test struct {
		fields  fields
		ctx     context.Context
		wantErr bool
		want    []*contextStatus
	}
	tests := map[string]test{
		"returns error when the GetCombinedStatus returns an error": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return nil, nil, errors.New("err")
				},
			}
			return test{
				fields: fields{
					client:        c,
					targetJobName: "target-job",
					owner:         "test-owner",
					repo:          "test-repo",
					ref:           "main",
				},
				wantErr: true,
			}
		}(),
		"returns error when the ListCheckRunsForRef returns an error": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return nil, nil, errors.New("error")
				},
			}
			return test{
				fields: fields{
					client:        c,
					targetJobName: "target-job",
					owner:         "test-owner",
					repo:          "test-repo",
					ref:           "main",
				},
				wantErr: true,
			}
		}(),
		"returns nil when the no error occurs": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []github.RepoStatus{
							{},
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{},
							{
								Name:   stringPtr("job-02"),
								Status: stringPtr("failure"),
							},
							{
								Name:       stringPtr("job-03"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunNeutralConclusion),
							},
							{
								Name:       stringPtr("job-04"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSuccessConclusion),
							},
							{
								Name:       stringPtr("job-05"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr("failure"),
							},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:        c,
					targetJobName: "target-job",
					owner:         "test-owner",
					repo:          "test-repo",
					ref:           "main",
				},
				wantErr: false,
				want: []*contextStatus{
					{
						Context: "job-01",
						State:   successState,
					},
					{
						Context: "job-02",
						State:   pendingState,
					},
					{
						Context: "job-03",
						State:   successState,
					},
					{
						Context: "job-04",
						State:   successState,
					},
					{
						Context: "job-05",
						State:   errorState,
					},
				},
			}
		}(),
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sv := &statusValidator{
				repo:          tt.fields.repo,
				owner:         tt.fields.owner,
				ref:           tt.fields.ref,
				targetJobName: tt.fields.targetJobName,
				client:        tt.fields.client,
			}
			got, err := sv.listStatuses(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("statusValidator.listStatuses() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got, want := len(got), len(tt.want); got != want {
				t.Errorf("statusValidator.listStatuses() length = %v, want %v", got, want)
			}
			for i := range tt.want {
				if !reflect.DeepEqual(got[i], tt.want[i]) {
					t.Errorf("statusValidator.listStatuses() - %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
