package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/upsidr/check-other-job-status/internal/github"
	"github.com/upsidr/check-other-job-status/internal/validators"
	"github.com/upsidr/check-other-job-status/internal/validators/status"
)

const defaultJobName = "check-other-job-status"

// These variables will be set by command line flags.
var (
	ghOwner             string
	ghRepo              string
	ghRef               string
	timeoutSecond       uint
	validateInvalSecond uint
	targetJobName       string
)

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate other github actions job",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			statusValidator, err := status.CreateValidator(github.NewClient(ctx, ghToken),
				status.WithTargetJob(targetJobName),
				status.WithGitHubOwnerAndRepo(ghOwner, ghRepo),
				status.WithGitHubRef(ghRef),
			)
			if err != nil {
				return fmt.Errorf("failed to create validator: %w", err)
			}
			return doValidateCmd(ctx, cmd, statusValidator)
		},
	}

	cmd.PersistentFlags().StringVarP(&targetJobName, "job", "j", defaultJobName, "set target job name")

	cmd.PersistentFlags().StringVarP(&ghOwner, "owner", "o", "", "set owner of github repository")
	cmd.MarkPersistentFlagRequired("owpner")

	cmd.PersistentFlags().StringVarP(&ghRepo, "repo", "r", "", "set github repository")
	cmd.MarkPersistentFlagRequired("repo")

	cmd.PersistentFlags().StringVar(&ghRef, "ref", "", "set ref of github repository. the ref can be a SHA, a branch name, or tag name")
	cmd.MarkPersistentFlagRequired("ref")

	cmd.PersistentFlags().UintVar(&timeoutSecond, "timeout", 600, "set validate timeout second")

	cmd.PersistentFlags().UintVar(&validateInvalSecond, "interval", 10, "set validate interval second")

	return cmd
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
				finishLog := debug(logger, "validator: "+v.Name())

				st, err := v.Validate(ctx)
				if err != nil {
					return fmt.Errorf("error occurs\tvalidator: %s, err: %v", v.Name(), err)
				}

				logger.Println(st.Detail())
				if st.IsSuccess() {
					successCnt++
				}
				finishLog()
			}
			if successCnt == len(vs) {
				logger.Println("all validations successful")
				return nil
			}
			logger.PrintErrln("validation failed")
		}
	}
}
