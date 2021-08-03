package mock

import (
	"context"

	"github.com/upsidr/check-other-job-status/internal/github"
)

type Client struct {
	GetCombinedStatusFunc   func(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error)
	ListCheckRunsForRefFunc func(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error)
}

func (c *Client) GetCombinedStatus(ctx context.Context, owner, repo, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
	return c.GetCombinedStatusFunc(ctx, owner, repo, ref, opts)
}

func (c *Client) ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
	return c.ListCheckRunsForRefFunc(ctx, owner, repo, ref, opts)
}

var (
	_ github.Client = &Client{}
)
