package status

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/upsidr/merge-gatekeeper/internal/github"
	"github.com/upsidr/merge-gatekeeper/internal/github/mock"
	"github.com/upsidr/merge-gatekeeper/internal/validators"
)

func stringPtr(str string) *string {
	return &str
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestCreateValidator(t *testing.T) {
	tests := map[string]struct {
		c       github.Client
		opts    []Option
		want    validators.Validator
		wantErr bool
	}{
		"returns Validator when option is not empty": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test-owner", "test-repo"),
				WithGitHubRef("sha"),
				WithSelfJob("job"),
				WithIgnoredJobs("job-01,job-02"),
			},
			want: &statusValidator{
				client:      &mock.Client{},
				owner:       "test-owner",
				repo:        "test-repo",
				ref:         "sha",
				selfJobName: "job",
				ignoredJobs: []string{"job-01", "job-02"},
			},
			wantErr: false,
		},
		"returns Validator when there are duplicate options": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test", "test-repo"),
				WithGitHubRef("sha"),
				WithGitHubRef("sha-01"),
				WithSelfJob("job"),
				WithSelfJob("job-01"),
			},
			want: &statusValidator{
				client:      &mock.Client{},
				owner:       "test",
				repo:        "test-repo",
				ref:         "sha-01",
				selfJobName: "job-01",
			},
			wantErr: false,
		},
		"returns Validator when invalid string is provided for ignored jobs": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test", "test-repo"),
				WithGitHubRef("sha"),
				WithGitHubRef("sha-01"),
				WithSelfJob("job"),
				WithSelfJob("job-01"),
				WithIgnoredJobs(","), // Malformed but handled
			},
			want: &statusValidator{
				client:      &mock.Client{},
				owner:       "test",
				repo:        "test-repo",
				ref:         "sha-01",
				selfJobName: "job-01",
				ignoredJobs: []string{}, // Not nil
			},
			wantErr: false,
		},
		"returns error when option is empty": {
			c:       &mock.Client{},
			want:    nil,
			wantErr: true,
		},
		"returns error when client is nil": {
			c: nil,
			opts: []Option{
				WithGitHubOwnerAndRepo("test", "test-repo"),
				WithGitHubRef("sha"),
				WithGitHubRef("sha-01"),
				WithSelfJob("job"),
				WithSelfJob("job-01"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := CreateValidator(tt.c, tt.opts...)
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

func TestName(t *testing.T) {
	tests := map[string]struct {
		c    github.Client
		opts []Option
		want string
	}{
		"Name returns the correct job name which gets overridden": {
			c: &mock.Client{},
			opts: []Option{
				WithGitHubOwnerAndRepo("test-owner", "test-repo"),
				WithGitHubRef("sha"),
				WithSelfJob("job"),
				WithIgnoredJobs("job-01,job-02"),
			},
			want: "job",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := CreateValidator(tt.c, tt.opts...)
			if err != nil {
				t.Errorf("Unexpected error with CreateValidator: %v", err)
				return
			}
			if tt.want != got.Name() {
				t.Errorf("Job name didn't match, want: %s, got: %v", tt.want, got.Name())
			}
		})
	}
}

func Test_statusValidator_Validate(t *testing.T) {
	type test struct {
		selfJobName string
		ignoredJobs []string
		client      github.Client
		ctx         context.Context
		wantErr     bool
		wantErrStr  string
		wantStatus  validators.Status
	}
	tests := map[string]test{
		"returns error when listGhaStatuses return an error": {
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return nil, nil, errors.New("err")
				},
			},
			wantErr:    true,
			wantStatus: nil,
			wantErrStr: "err",
		},
		"returns succeeded status and nil when there is no job": {
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{},
				completeJobs: []string{},
				errJobs:      []string{},
			},
		},
		"returns succeeded status and nil when there is one job, which is itself": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState), // should be irrelevant
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{},
				completeJobs: []string{},
				errJobs:      []string{},
			},
		},
		"returns failed status and nil when there is one job": {
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    false,
				totalJobs:    []string{"job"},
				completeJobs: []string{},
				errJobs:      []string{},
			},
		},
		"returns error when there is a failed job": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(errorState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: true,
			wantErrStr: (&status{
				totalJobs: []string{
					"job-01", "job-02",
				},
				completeJobs: []string{
					"job-01",
				},
				errJobs: []string{
					"job-02",
				},
			}).Detail(),
		},
		"returns error when there is a failed job with failure state": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(failureState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: true,
			wantErrStr: (&status{
				totalJobs: []string{
					"job-01", "job-02",
				},
				completeJobs: []string{
					"job-01",
				},
				errJobs: []string{
					"job-02",
				},
			}).Detail(),
		},
		"returns failed status and nil when successful job count is less than total": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(pendingState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded: false,
				totalJobs: []string{
					"job-01",
					"job-02",
				},
				completeJobs: []string{
					"job-01",
				},
				errJobs: []string{},
			},
		},
		"returns succeeded status and nil when validation is success": {
			selfJobName: "self-job",
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded: true,
				totalJobs: []string{
					"job-01",
					"job-02",
				},
				completeJobs: []string{
					"job-01",
					"job-02",
				},
				errJobs: []string{},
			},
		},
		"returns succeeded status and nil when only an ignored job is failing": {
			selfJobName: "self-job",
			ignoredJobs: []string{"job-02", "job-03"}, // String input here should be already TrimSpace'd
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(errorState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{"job-01"},
				completeJobs: []string{"job-01"},
				errJobs:      []string{},
			},
		},
		"returns succeeded status and nil when only an ignored job is failing, with failure state": {
			selfJobName: "self-job",
			ignoredJobs: []string{"job-02", "job-03"},
			client: &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-02"),
								State:   stringPtr(failureState),
							},
							{
								Context: stringPtr("self-job"),
								State:   stringPtr(pendingState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{}, nil, nil
				},
			},
			wantErr: false,
			wantStatus: &status{
				succeeded:    true,
				totalJobs:    []string{"job-01"},
				completeJobs: []string{"job-01"},
				errJobs:      []string{},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sv := &statusValidator{
				selfJobName: tt.selfJobName,
				ignoredJobs: tt.ignoredJobs,
				client:      tt.client,
			}
			got, err := sv.Validate(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("statusValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.wantErrStr {
					t.Errorf("statusValidator.Validate() error.Error() = %s, wantErrStr %s", err.Error(), tt.wantErrStr)
				}
			}
			if !reflect.DeepEqual(got, tt.wantStatus) {
				t.Errorf("statusValidator.Validate() status = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

func Test_statusValidator_listStatues(t *testing.T) {
	type fields struct {
		repo        string
		owner       string
		ref         string
		selfJobName string
		client      github.Client
	}
	type test struct {
		fields  fields
		ctx     context.Context
		wantErr bool
		want    []*ghaStatus
	}
	tests := map[string]test{
		"succeeds to get job statuses even if the same job exists": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							// The first element here is the latest state.
							{
								Context: stringPtr("job-01"),
								State:   stringPtr(successState),
							},
							{
								Context: stringPtr("job-01"), // Same as above job name, and thus should be disregarded as old job status.
								State:   stringPtr(errorState),
							},
						},
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							// The first element here is the latest state.
							{
								Name:   stringPtr("job-02"),
								Status: stringPtr("failure"),
							},
							{
								Name:       stringPtr("job-02"), // Same as above job name, and thus should be disregarded as old job status.
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunNeutralConclusion),
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
							{
								Name:       stringPtr("job-06"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSkipConclusion),
							},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want: []*ghaStatus{
					{
						Job:   "job-01",
						State: successState,
					},
					{
						Job:   "job-02",
						State: pendingState,
					},
					{
						Job:   "job-03",
						State: successState,
					},
					{
						Job:   "job-04",
						State: successState,
					},
					{
						Job:   "job-05",
						State: errorState,
					},
				},
			}
		}(),
		"returns error when the GetCombinedStatus returns an error": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return nil, nil, errors.New("err")
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: true,
			}
		}(),
		"returns error when the GetCombinedStatus response is invalid": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
							{},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
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
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: true,
			}
		}(),
		"returns error when the ListCheckRunsForRef response is invalid": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					return &github.ListCheckRunsResults{
						CheckRuns: []*github.CheckRun{
							{},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: true,
			}
		}(),
		"returns nil when no error occurs": func() test {
			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					return &github.CombinedStatus{
						Statuses: []*github.RepoStatus{
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
							{
								Name:       stringPtr("job-06"),
								Status:     stringPtr(checkRunCompletedStatus),
								Conclusion: stringPtr(checkRunSkipConclusion),
							},
						},
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want: []*ghaStatus{
					{
						Job:   "job-01",
						State: successState,
					},
					{
						Job:   "job-02",
						State: pendingState,
					},
					{
						Job:   "job-03",
						State: successState,
					},
					{
						Job:   "job-04",
						State: successState,
					},
					{
						Job:   "job-05",
						State: errorState,
					},
				},
			}
		}(),
		"succeeds to retrieve 100 statuses": func() test {
			num_statuses := 100
			statuses := make([]*github.RepoStatus, num_statuses)
			checkRuns := make([]*github.CheckRun, num_statuses)
			expectedGhaStatuses := make([]*ghaStatus, num_statuses)
			for i := 0; i < num_statuses; i++ {
				statuses[i] = &github.RepoStatus{
					Context: stringPtr(fmt.Sprintf("job-%d", i)),
					State:   stringPtr(successState),
				}

				checkRuns[i] = &github.CheckRun{
					Name:       stringPtr(fmt.Sprintf("job-%d", i)),
					Status:     stringPtr(checkRunCompletedStatus),
					Conclusion: stringPtr(checkRunNeutralConclusion),
				}

				expectedGhaStatuses[i] = &ghaStatus{
					Job:   fmt.Sprintf("job-%d", i),
					State: successState,
				}
			}

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					max := min(opts.Page*opts.PerPage, len(statuses))
					sts := statuses[(opts.Page-1)*opts.PerPage : max]
					l := len(sts)
					return &github.CombinedStatus{
						Statuses:   sts,
						TotalCount: &l,
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					max := min(opts.ListOptions.Page*opts.ListOptions.PerPage, len(checkRuns))
					sts := checkRuns[(opts.ListOptions.Page-1)*opts.ListOptions.PerPage : max]
					l := len(sts)
					return &github.ListCheckRunsResults{
						CheckRuns: checkRuns,
						Total:     &l,
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want:    expectedGhaStatuses,
			}
		}(),
		"succeeds to retrieve 162 statuses": func() test {
			num_statuses := 162
			statuses := make([]*github.RepoStatus, num_statuses)
			checkRuns := make([]*github.CheckRun, num_statuses)
			expectedGhaStatuses := make([]*ghaStatus, num_statuses)
			for i := 0; i < num_statuses; i++ {
				statuses[i] = &github.RepoStatus{
					Context: stringPtr(fmt.Sprintf("job-%d", i)),
					State:   stringPtr(successState),
				}

				checkRuns[i] = &github.CheckRun{
					Name:       stringPtr(fmt.Sprintf("job-%d", i)),
					Status:     stringPtr(checkRunCompletedStatus),
					Conclusion: stringPtr(checkRunNeutralConclusion),
				}

				expectedGhaStatuses[i] = &ghaStatus{
					Job:   fmt.Sprintf("job-%d", i),
					State: successState,
				}
			}

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					max := min(opts.Page*opts.PerPage, len(statuses))
					sts := statuses[(opts.Page-1)*opts.PerPage : max]
					l := len(sts)
					return &github.CombinedStatus{
						Statuses:   sts,
						TotalCount: &l,
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					max := min(opts.ListOptions.Page*opts.ListOptions.PerPage, len(checkRuns))
					sts := checkRuns[(opts.ListOptions.Page-1)*opts.ListOptions.PerPage : max]
					l := len(sts)
					return &github.ListCheckRunsResults{
						CheckRuns: checkRuns,
						Total:     &l,
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want:    expectedGhaStatuses,
			}
		}(),
		"succeeds to retrieve 587 statuses": func() test {
			num_statuses := 587
			statuses := make([]*github.RepoStatus, num_statuses)
			checkRuns := make([]*github.CheckRun, num_statuses)
			expectedGhaStatuses := make([]*ghaStatus, num_statuses)
			for i := 0; i < num_statuses; i++ {
				statuses[i] = &github.RepoStatus{
					Context: stringPtr(fmt.Sprintf("job-%d", i)),
					State:   stringPtr(successState),
				}

				checkRuns[i] = &github.CheckRun{
					Name:       stringPtr(fmt.Sprintf("job-%d", i)),
					Status:     stringPtr(checkRunCompletedStatus),
					Conclusion: stringPtr(checkRunNeutralConclusion),
				}

				expectedGhaStatuses[i] = &ghaStatus{
					Job:   fmt.Sprintf("job-%d", i),
					State: successState,
				}
			}

			c := &mock.Client{
				GetCombinedStatusFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
					max := min(opts.Page*opts.PerPage, len(statuses))
					sts := statuses[(opts.Page-1)*opts.PerPage : max]
					l := len(sts)
					return &github.CombinedStatus{
						Statuses:   sts,
						TotalCount: &l,
					}, nil, nil
				},
				ListCheckRunsForRefFunc: func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
					max := min(opts.ListOptions.Page*opts.ListOptions.PerPage, len(checkRuns))
					sts := checkRuns[(opts.ListOptions.Page-1)*opts.ListOptions.PerPage : max]
					l := len(sts)
					return &github.ListCheckRunsResults{
						CheckRuns: checkRuns,
						Total:     &l,
					}, nil, nil
				},
			}
			return test{
				fields: fields{
					client:      c,
					selfJobName: "self-job",
					owner:       "test-owner",
					repo:        "test-repo",
					ref:         "main",
				},
				wantErr: false,
				want:    expectedGhaStatuses,
			}
		}(),
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sv := &statusValidator{
				repo:        tt.fields.repo,
				owner:       tt.fields.owner,
				ref:         tt.fields.ref,
				selfJobName: tt.fields.selfJobName,
				client:      tt.fields.client,
			}
			got, err := sv.listGhaStatuses(tt.ctx)
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
