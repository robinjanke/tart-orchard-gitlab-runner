package cleanup

import (
	"log"

	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/flags"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/gitlab"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/orchard"
	"github.com/spf13/cobra"
)

var orchardFlags flags.OrchardFlags

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete the Orchard VM after the job finishes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cleanupVM(cmd); err != nil {
				return gitlab.NewSystemFailureError(err)
			}
			return nil
		},
	}

	orchardFlags.RegisterConnection(cmd)
	return cmd
}

func cleanupVM(cmd *cobra.Command) error {
	cfg, err := orchardFlags.Config()
	if err != nil {
		return err
	}

	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return err
	}

	client, err := orchard.NewClient(cfg)
	if err != nil {
		return err
	}

	if err := orchard.DeleteVM(cmd.Context(), client, gitLabEnv.VirtualMachineID()); err != nil {
		// Already gone (e.g. never created / previous cleanup) — not a job failure.
		if orchard.IsNotFound(err) {
			log.Printf("Orchard VM %q already gone; skipping delete", gitLabEnv.VirtualMachineID())
			return nil
		}
		log.Printf("Failed to delete VM: %v", err)
		return err
	}
	return nil
}
