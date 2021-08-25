package status

import (
	"context"
	"errors"
	"fmt"

	"github.com/upsidr/merge-gatekeeper/internal/github"
	"github.com/upsidr/merge-gatekeeper/internal/multierror"
	"github.com/upsidr/merge-gatekeeper/internal/validators"
)

const (
	successState = "success"
	errorState   = "error"
	pendingState = "pending"
)

// NOTE: https://docs.github.com/en/rest/reference/checks
const (
	checkRunCompletedStatus = "completed"
)
const (
	checkRunNeutralConclusion = "neutral"
	checkRunSuccessConclusion = "success"
)

var (
	ErrInvalidCombinedStatusResponse = errors.New("github combined status response is invalid")
	ErrInvalidCheckRunResponse       = errors.New("github checkRun response is invalid")
)

type ghaStatus struct {
	Job   string
	State string
}

type statusValidator struct {
	repo          string
	owner         string
	ref           string
	targetJobName string
	client        github.Client
}

func CreateValidator(c github.Client, opts ...Option) (validators.Validator, error) {
	sv := &statusValidator{
		client: c,
	}
	for _, opt := range opts {
		opt(sv)
	}
	if err := sv.validateFields(); err != nil {
		return nil, err
	}
	return sv, nil
}

func (sv *statusValidator) Name() string {
	return sv.targetJobName
}

func (sv *statusValidator) validateFields() error {
	errs := make(multierror.Errors, 0, 6)

	if len(sv.repo) == 0 {
		errs = append(errs, errors.New("repository name is empty"))
	}
	if len(sv.owner) == 0 {
		errs = append(errs, errors.New("repository owner is empty"))
	}
	if len(sv.ref) == 0 {
		errs = append(errs, errors.New("reference of repository is empty"))
	}
	if len(sv.targetJobName) == 0 {
		errs = append(errs, errors.New("target job name is empty"))
	}
	if sv.client == nil {
		errs = append(errs, errors.New("github client is empty"))
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (sv *statusValidator) Validate(ctx context.Context) (validators.Status, error) {
	ghaStatuses, err := sv.listGhaStatuses(ctx)
	if err != nil {
		return nil, err
	}

	st := &status{
		totalJobs:    make([]string, 0, len(ghaStatuses)),
		completeJobs: make([]string, 0, len(ghaStatuses)),
		succeeded:    true,
	}

	switch len(ghaStatuses) {
	case 0:
		return st, nil

	// When there is no job than this validation job.
	case 1:
		st.totalJobs = append(st.totalJobs, ghaStatuses[0].Job)
		return st, nil
	}

	var successCnt int
	for _, ghaStatus := range ghaStatuses {
		if ghaStatus.Job != sv.targetJobName {
			st.totalJobs = append(st.totalJobs, ghaStatus.Job)

			if ghaStatus.State != successState {
				continue
			}
			st.completeJobs = append(st.completeJobs, ghaStatus.Job)
			successCnt++
		}
	}
	if len(ghaStatuses)-1 != successCnt {
		st.succeeded = false
		return st, nil
	}

	return st, nil
}

func (sv *statusValidator) listGhaStatuses(ctx context.Context) ([]*ghaStatus, error) {
	combined, _, err := sv.client.GetCombinedStatus(ctx, sv.owner, sv.repo, sv.ref, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	ghaStatuses := make([]*ghaStatus, 0, len(combined.Statuses))
	for _, s := range combined.Statuses {
		if s.Context == nil || s.State == nil {
			return nil, fmt.Errorf("%w context: %v, status: %v", ErrInvalidCombinedStatusResponse, s.Context, s.State)
		}
		ghaStatuses = append(ghaStatuses, &ghaStatus{
			Job:   *s.Context,
			State: *s.State,
		})
	}

	runResult, _, err := sv.client.ListCheckRunsForRef(ctx, sv.owner, sv.repo, sv.ref, &github.ListCheckRunsOptions{})
	if err != nil {
		return nil, err
	}

	for _, run := range runResult.CheckRuns {
		if run.Name == nil || run.Status == nil {
			return nil, fmt.Errorf("%w name: %v, status: %v", ErrInvalidCheckRunResponse, run.Name, run.Status)
		}
		ghaStatus := &ghaStatus{
			Job: *run.Name,
		}
		if *run.Status != checkRunCompletedStatus {
			ghaStatus.State = pendingState
			ghaStatuses = append(ghaStatuses, ghaStatus)
			continue
		}

		switch *run.Conclusion {
		case checkRunNeutralConclusion, checkRunSuccessConclusion:
			ghaStatus.State = successState
		default:
			ghaStatus.State = errorState
		}
		ghaStatuses = append(ghaStatuses, ghaStatus)
	}

	return ghaStatuses, nil
}
