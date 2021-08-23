package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/upsidr/check-other-job-status/internal/github"
	"github.com/upsidr/check-other-job-status/internal/validators"
	"github.com/upsidr/check-other-job-status/internal/validators/status"
)

const defaultJobName = "check-other-job-status"

// These variables will be set by command line flags.
var (
	ghRepo              string // e.g) upsidr/check-other-job-status
	ghRef               string
	timeoutSecond       uint
	validateInvalSecond uint
	targetJobName       string
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

			statusValidator, err := status.CreateValidator(github.NewClient(ctx, ghToken),
				status.WithTargetJob(targetJobName),
				status.WithGitHubOwnerAndRepo(owner, repo),
				status.WithGitHubRef(ghRef),
			)
			if err != nil {
				return fmt.Errorf("failed to create validator: %w", err)
			}
			return doValidateCmd(ctx, cmd, statusValidator)
		},
	}

	cmd.PersistentFlags().StringVarP(&targetJobName, "job", "j", defaultJobName, "set target job name")

	cmd.PersistentFlags().StringVarP(&ghRepo, "repo", "r", "", "set github repository")
	cmd.MarkPersistentFlagRequired("repo")

	cmd.PersistentFlags().StringVar(&ghRef, "ref", "", "set ref of github repository. the ref can be a SHA, a branch name, or tag name")
	cmd.MarkPersistentFlagRequired("ref")

	cmd.PersistentFlags().UintVar(&timeoutSecond, "timeout", 600, "set validate timeout second")

	cmd.PersistentFlags().UintVar(&validateInvalSecond, "interval", 10, "set validate interval second")

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
	logger.Printf("start %s processing....\n", name)
	return func() {
		logger.Printf("finish %s processing\n", name)
	}
}

func doValidateCmd(ctx context.Context, logger logger, vs ...validators.Validator) error {
	timeoutT := time.NewTicker(time.Duration(timeoutSecond) * time.Second)
	defer timeoutT.Stop()

	invalT := time.NewTicker(time.Duration(validateInvalSecond) * time.Second)
	defer invalT.Stop()

	defer debug(logger, "validation loop")()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutT.C:
			return errors.New("validation timed out")
		case <-invalT.C:
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
			if successCnt == len(vs) {
				logger.Println("all validations successful")
				return nil
			}
			logger.PrintErrln("validation failed")
		}
	}
}

func validate(ctx context.Context, v validators.Validator, logger logger) (bool, error) {
	defer debug(logger, "validator: "+v.Name())()

	st, err := v.Validate(ctx)
	if err != nil {
		return false, fmt.Errorf("error occurs\tvalidator: %s, err: %v", v.Name(), err)
	}

	logger.Println(st.Detail())

	if !st.IsSuccess() {
		return false, nil
	}
	return true, nil
}
