package status

import (
	"context"
	"errors"

	"github.com/upsidr/check-other-job-status/internal/github"
	"github.com/upsidr/check-other-job-status/internal/validators"
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

type contextStatus struct {
	Context string
	State   string
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

func (sv *statusValidator) validateFields() error {
	errs := make(errs, 0, 6)

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

func (sv *statusValidator) Validate(ctx context.Context) error {
	statuses, err := sv.listStatuses(ctx)
	if err != nil {
		return err
	}

	// When there is no other job than this validation job.
	if len(statuses) <= 1 {
		return nil
	}

	var successJobCnt int
	for _, status := range statuses {
		if status.Context != sv.targetJobName && status.State == successState {
			successJobCnt++
		}
	}
	if len(statuses)-1 != successJobCnt {
		return validators.ErrValidate
	}
	return nil
}

func (sv *statusValidator) listStatuses(ctx context.Context) ([]*contextStatus, error) {
	combined, _, err := sv.client.GetCombinedStatus(ctx, sv.owner, sv.repo, sv.ref, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	contextStatuses := make([]*contextStatus, 0, len(combined.Statuses))
	for _, s := range combined.Statuses {
		if s.Context == nil || s.State == nil {
			continue
		}
		contextStatuses = append(contextStatuses, &contextStatus{
			Context: *s.Context,
			State:   *s.State,
		})
	}

	runResult, _, err := sv.client.ListCheckRunsForRef(ctx, sv.owner, sv.repo, sv.ref, &github.ListCheckRunsOptions{})
	if err != nil {
		return nil, err
	}

	for _, run := range runResult.CheckRuns {
		if run.Name == nil || run.Status == nil {
			continue
		}
		contextStatus := &contextStatus{
			Context: *run.Name,
		}
		if *run.Status != checkRunCompletedStatus {
			contextStatus.State = pendingState
			contextStatuses = append(contextStatuses, contextStatus)
			continue
		}

		switch *run.Conclusion {
		case checkRunNeutralConclusion, checkRunSuccessConclusion:
			contextStatus.State = successState
		default:
			contextStatus.State = errorState
		}
		contextStatuses = append(contextStatuses, contextStatus)
	}

	return contextStatuses, nil
}
