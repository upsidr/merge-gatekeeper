package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
)

type (
	ListOptions    = github.ListOptions
	CombinedStatus = github.CombinedStatus
	RepoStatus     = github.RepoStatus
	Response       = github.Response
)

type (
	CheckRun             = github.CheckRun
	ListCheckRunsOptions = github.ListCheckRunsOptions
	ListCheckRunsResults = github.ListCheckRunsResults
)

type Client interface {
	GetCombinedStatus(ctx context.Context, owner, repo, ref string, opts *ListOptions) (*CombinedStatus, *Response, error)
	ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *ListCheckRunsOptions) (*ListCheckRunsResults, *Response, error)
}

type client struct {
	ghc *github.Client
	maxRetries int
	retryDelay time.Duration
}

func NewClient(ctx context.Context, token string) Client {
	return &client{
		ghc: github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: token,
			},
		))),
		maxRetries: 5,
		retryDelay: 1 * time.Second,
	}
}

func (c *client) GetCombinedStatus(ctx context.Context, owner, repo, ref string, opts *ListOptions) (*CombinedStatus, *Response, error) {
	var statusResp *CombinedStatus
	var resp *Response
	var err error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		statusResp, resp, err = c.ghc.Repositories.GetCombinedStatus(ctx, owner, repo, ref, opts)
		if err == nil {
			return statusResp, resp, nil
		}

		// Check if context is canceled or deadline exceeded before retrying
		if ctx.Err() != nil {
			return nil, resp, fmt.Errorf("context error while getting combined status: %w", ctx.Err())
		}
		
		// Don't retry on deadline exceeded errors
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, resp, err
		}

		// Only retry on 5xx server errors
		if resp != nil && (resp.StatusCode < 500 || resp.StatusCode > 599) {
			return statusResp, resp, err
		}

		// Wait with exponential backoff before retrying
		if attempt < c.maxRetries-1 {
			backoffDuration := c.retryDelay * time.Duration(1<<attempt)
			select {
			case <-ctx.Done():
				return nil, resp, ctx.Err()
			case <-time.After(backoffDuration):
				// Continue with retry
			}
		}
	}

	return nil, resp, fmt.Errorf("failed to get combined status after %d retries: %w", c.maxRetries, err)
}

func (c *client) ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *ListCheckRunsOptions) (*ListCheckRunsResults, *Response, error) {
	var checksResp *ListCheckRunsResults
	var resp *Response
	var err error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		checksResp, resp, err = c.ghc.Checks.ListCheckRunsForRef(ctx, owner, repo, ref, opts)
		if err == nil {
			return checksResp, resp, nil
		}

		// Check if context is canceled or deadline exceeded before retrying
		if ctx.Err() != nil {
			return nil, resp, fmt.Errorf("context error while listing check runs: %w", ctx.Err())
		}
		
		// Don't retry on deadline exceeded errors
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, resp, err
		}

		// Only retry on 5xx server errors
		if resp != nil && (resp.StatusCode < 500 || resp.StatusCode > 599) {
			return checksResp, resp, err
		}

		// Wait with exponential backoff before retrying
		if attempt < c.maxRetries-1 {
			backoffDuration := c.retryDelay * time.Duration(1<<attempt)
			select {
			case <-ctx.Done():
				return nil, resp, ctx.Err()
			case <-time.After(backoffDuration):
				// Continue with retry
			}
		}
	}

	return nil, resp, fmt.Errorf("failed to list check runs after %d retries: %w", c.maxRetries, err)
}
