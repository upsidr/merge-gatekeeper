package cli

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/upsidr/check-other-job-status/internal/github"
	"github.com/upsidr/check-other-job-status/internal/validators"
	"github.com/upsidr/check-other-job-status/internal/validators/status"
)

const defaultJobName = "check-other-job-status"

// Tease variables will be set by command line flags.
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

			statusValidator := status.CreateValidator(github.NewClient(ctx, ghToken),
				status.WithTargetJob(targetJobName),
				status.WithGitHubOwnerAndRepo(ghOwner, ghRepo),
				status.WithGitHubRef(ghRef),
			)
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

func doValidateCmd(ctx context.Context, logger logger, vs ...validators.Validator) error {
	timeoutT := time.NewTicker(time.Duration(timeoutSecond) * time.Second)
	defer timeoutT.Stop()

	invalT := time.NewTicker(time.Duration(validateInvalSecond) * time.Second)
	defer invalT.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutT.C:
			return errors.New("validation is timeout")
		case <-invalT.C:
			var successCnt int
			for _, validator := range vs {
				err := validator.Validate(ctx)
				if err != nil {
					if !errors.Is(err, validators.ErrValidate) {
						return err
					}
					logger.PrintErrln(err)
					break
				} else {
					successCnt++
				}
			}
			if successCnt == len(vs) {
				return nil
			}
		}
	}
}
