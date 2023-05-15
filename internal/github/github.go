package github

import (
	"context"
	"net/http"

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
}

func NewClient(ctx context.Context, token string) Client {
	oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	))

	return &client{
		ghc: github.NewClient(&http.Client{
			Transport: &RetryTransport{
				Transport: oauthClient.Transport,
				Retries:   3,
			},
		}),
	}
}

func (c *client) GetCombinedStatus(ctx context.Context, owner, repo, ref string, opts *ListOptions) (*CombinedStatus, *Response, error) {
	return c.ghc.Repositories.GetCombinedStatus(ctx, owner, repo, ref, opts)
}

func (c *client) ListCheckRunsForRef(ctx context.Context, owner, repo, ref string, opts *ListCheckRunsOptions) (*ListCheckRunsResults, *Response, error) {
	return c.ghc.Checks.ListCheckRunsForRef(ctx, owner, repo, ref, opts)
}
