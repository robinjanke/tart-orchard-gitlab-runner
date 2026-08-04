package commands

import (
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/commands/cleanup"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/commands/config"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/commands/prepare"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/commands/run"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gitlab-orchard-executor",
		Short:         "GitLab Runner custom executor for Orchard macOS VMs",
		Version:       version.FullVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(config.NewCommand())
	cmd.AddCommand(prepare.NewCommand())
	cmd.AddCommand(run.NewCommand())
	cmd.AddCommand(cleanup.NewCommand())

	return cmd
}
