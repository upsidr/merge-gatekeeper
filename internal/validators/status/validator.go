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
	failureState = "failure"
	pendingState = "pending"
)

// NOTE: https://docs.github.com/en/rest/reference/checks
const (
	checkRunCompletedStatus = "completed"
)
const (
	checkRunNeutralConclusion = "neutral"
	checkRunSuccessConclusion = "success"
	checkRunSkipConclusion    = "skipped"
)

const (
	maxStatusesPerPage  = 100
	maxCheckRunsPerPage = 100
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
	repo        string
	owner       string
	ref         string
	selfJobName string
	ignoredJobs []string
	client      github.Client
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
	return sv.selfJobName
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
	if len(sv.selfJobName) == 0 {
		errs = append(errs, errors.New("self job name is empty"))
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
		errJobs:      make([]string, 0, len(ghaStatuses)/2),
		succeeded:    true,
	}

	var successCnt int
	for _, ghaStatus := range ghaStatuses {
		var toIgnore bool
		for _, ignored := range sv.ignoredJobs {
			if ghaStatus.Job == ignored {
				toIgnore = true
				break
			}
		}

		// Ignored jobs and this job itself should be considered as success regardless of their statuses.
		if toIgnore || ghaStatus.Job == sv.selfJobName {
			successCnt++
			continue
		}

		st.totalJobs = append(st.totalJobs, ghaStatus.Job)

		switch ghaStatus.State {
		case successState:
			st.completeJobs = append(st.completeJobs, ghaStatus.Job)
			successCnt++
		case errorState, failureState:
			st.errJobs = append(st.errJobs, ghaStatus.Job)
		}
	}
	if len(st.errJobs) != 0 {
		return nil, errors.New(st.Detail())
	}

	if len(ghaStatuses) != successCnt {
		st.succeeded = false
		return st, nil
	}

	return st, nil
}

func (sv *statusValidator) getCombinedStatus(ctx context.Context) ([]*github.RepoStatus, error) {
	var combined []*github.RepoStatus
	page := 1
	for {
		c, _, err := sv.client.GetCombinedStatus(ctx, sv.owner, sv.repo, sv.ref, &github.ListOptions{PerPage: maxStatusesPerPage, Page: page})
		if err != nil {
			return nil, err
		}
		combined = append(combined, c.Statuses...)
		if c.GetTotalCount() < maxStatusesPerPage {
			break
		}
		page++
	}
	return combined, nil
}

func (sv *statusValidator) listCheckRunsForRef(ctx context.Context) ([]*github.CheckRun, error) {
	var runResults []*github.CheckRun
	page := 1
	for {
		cr, _, err := sv.client.ListCheckRunsForRef(ctx, sv.owner, sv.repo, sv.ref, &github.ListCheckRunsOptions{ListOptions: github.ListOptions{
			Page:    page,
			PerPage: maxCheckRunsPerPage,
		}})
		if err != nil {
			return nil, err
		}
		runResults = append(runResults, cr.CheckRuns...)
		if cr.GetTotal() < maxCheckRunsPerPage {
			break
		}
		page++
	}
	return runResults, nil
}

func (sv *statusValidator) listGhaStatuses(ctx context.Context) ([]*ghaStatus, error) {
	combined, err := sv.getCombinedStatus(ctx)
	if err != nil {
		return nil, err
	}

	// Because multiple jobs with the same name may exist when jobs are created dynamically by third-party tools, etc.,
	// only the latest job should be managed.
	currentJobs := make(map[string]struct{})

	ghaStatuses := make([]*ghaStatus, 0, len(combined))
	for _, s := range combined {
		if s.Context == nil || s.State == nil {
			return nil, fmt.Errorf("%w context: %v, status: %v", ErrInvalidCombinedStatusResponse, s.Context, s.State)
		}
		if _, ok := currentJobs[*s.Context]; ok {
			continue
		}
		currentJobs[*s.Context] = struct{}{}

		ghaStatuses = append(ghaStatuses, &ghaStatus{
			Job:   *s.Context,
			State: *s.State,
		})
	}

	runResults, err := sv.listCheckRunsForRef(ctx)
	if err != nil {
		return nil, err
	}

	for _, run := range runResults {
		if run.Name == nil || run.Status == nil {
			return nil, fmt.Errorf("%w name: %v, status: %v", ErrInvalidCheckRunResponse, run.Name, run.Status)
		}
		if _, ok := currentJobs[*run.Name]; ok {
			continue
		}
		currentJobs[*run.Name] = struct{}{}

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
		case checkRunSkipConclusion:
			continue
		default:
			ghaStatus.State = errorState
		}
		ghaStatuses = append(ghaStatuses, ghaStatus)
	}

	return ghaStatuses, nil
}
