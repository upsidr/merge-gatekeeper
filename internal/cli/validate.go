package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/upsidr/merge-gatekeeper/internal/github"
	"github.com/upsidr/merge-gatekeeper/internal/ticker"
	"github.com/upsidr/merge-gatekeeper/internal/validators"
	"github.com/upsidr/merge-gatekeeper/internal/validators/status"
)

const defaultSelfJobName = "merge-gatekeeper"

// These variables will be set by command line flags.
var (
	ghRepo              string // e.g) upsidr/merge-gatekeeper
	ghRef               string
	timeoutSecond       uint
	validateInvalSecond uint
	selfJobName         string
	ignoredJobs         string
	githubClientRetry   int
)

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate other github actions job",
		PreRun: func(cmd *cobra.Command, args []string) {
			str := os.Getenv("GITHUB_REPOSITORY")
			if len(str) != 0 {
				ghRepo = str
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			owner, repo := ownerAndRepository(ghRepo)
			if len(owner) == 0 || len(repo) == 0 {
				return fmt.Errorf("github owner or repository is empty. owner: %s, repository: %s", owner, repo)
			}

			t := http.DefaultTransport
			if githubClientRetry > 0 {
				t = github.NewRetryTransport(githubClientRetry)
			}
			statusValidator, err := status.CreateValidator(
				github.NewClient(ctx, ghToken, github.WithTransport(t)),
				status.WithSelfJob(selfJobName),
				status.WithGitHubOwnerAndRepo(owner, repo),
				status.WithGitHubRef(ghRef),
				status.WithIgnoredJobs(ignoredJobs),
			)
			if err != nil {
				return fmt.Errorf("failed to create validator: %w", err)
			}

			cmd.SilenceUsage = true
			return doValidateCmd(ctx, cmd, statusValidator)
		},
	}

	cmd.PersistentFlags().StringVarP(&selfJobName, "self", "s", defaultSelfJobName, "set self job name")

	cmd.PersistentFlags().StringVarP(&ghRepo, "repo", "r", "", "set github repository")

	cmd.PersistentFlags().StringVar(&ghRef, "ref", "", "set ref of github repository. the ref can be a SHA, a branch name, or tag name")
	cmd.MarkPersistentFlagRequired("ref")

	cmd.PersistentFlags().UintVar(&timeoutSecond, "timeout", 600, "set validate timeout second")
	cmd.PersistentFlags().UintVar(&validateInvalSecond, "interval", 10, "set validate interval second")

	cmd.PersistentFlags().StringVarP(&ignoredJobs, "ignored", "i", "", "set ignored jobs (comma-separated list)")

	cmd.PersistentFlags().IntVar(&githubClientRetry, "github-client-retry", 0, "set retry count for GitHub client")

	return cmd
}

func ownerAndRepository(str string) (owner string, repo string) {
	sp := strings.Split(str, "/")
	switch len(sp) {
	case 0:
		return "", ""
	case 1:
		return sp[0], ""
	case 2:
		return sp[0], sp[1]
	default:
		return sp[0], strings.Join(sp[1:], "/")
	}
}

func debug(logger logger, name string) func() {
	logger.Printf("Start processing %s....\n", name)
	return func() {
		logger.Printf("Finish %s processing.\n", name)
	}
}

func doValidateCmd(ctx context.Context, logger logger, vs ...validators.Validator) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecond)*time.Second)
	defer cancel()

	invalT := ticker.NewInstantTicker(time.Duration(validateInvalSecond) * time.Second)
	defer invalT.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-invalT.C():
			var successCnt int
			for _, v := range vs {
				ok, err := validate(ctx, v, logger)
				if err != nil {
					return err
				}
				if ok {
					successCnt++
				}
			}
			if successCnt != len(vs) {
				logger.PrintErrln("")
				logger.PrintErrln("  WARNING: Validation is yet to be completed. This is most likely due to some other jobs still running.")
				logger.PrintErrf("           Waiting for %d seconds before retrying.\n\n", validateInvalSecond)
				break
			}

			logger.Println("All validations were successful!")
			return nil
		}
	}
}

func validate(ctx context.Context, v validators.Validator, logger logger) (bool, error) {
	defer debug(logger, "validator: "+v.Name())()

	st, err := v.Validate(ctx)
	if err != nil {
		return false, fmt.Errorf("validation failed, err: %v", err)
	}

	logger.Println(st.Detail())

	if !st.IsSuccess() {
		return false, nil
	}
	return true, nil
}
