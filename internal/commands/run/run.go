package run

import (
	"errors"

	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/flags"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/gitlab"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/orchard"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/sshutil"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var orchardFlags flags.OrchardFlags

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <path-to-script-file>",
		Short: "Run a GitLab script inside the Orchard VM",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runScriptInsideVM,
	}

	orchardFlags.RegisterConnection(cmd)
	cmd.Flags().StringVar(&orchardFlags.SSHUsername, "ssh-username", orchard.DefaultConfig().SSHUsername, "SSH username")
	cmd.Flags().StringVar(&orchardFlags.SSHPassword, "ssh-password", orchard.DefaultConfig().SSHPassword, "SSH password")
	cmd.Flags().Uint16Var(&orchardFlags.SSHPort, "ssh-port", orchard.DefaultConfig().SSHPort, "SSH port")
	cmd.Flags().StringVar(&orchardFlags.Shell, "shell", orchard.DefaultConfig().Shell, "Shell to run scripts")

	return cmd
}

func runScriptInsideVM(cmd *cobra.Command, args []string) error {
	cfg, err := orchardFlags.Config()
	if err != nil {
		return gitlab.NewSystemFailureError(err)
	}

	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return gitlab.NewSystemFailureError(err)
	}

	client, err := orchard.NewClient(cfg)
	if err != nil {
		return gitlab.NewSystemFailureError(err)
	}

	sshClient, err := sshutil.DialVM(
		cmd.Context(),
		client,
		gitLabEnv.VirtualMachineID(),
		cfg.SSHPort,
		cfg.PortForwardWait,
		cfg.SSHUsername,
		cfg.SSHPassword,
	)
	if err != nil {
		return gitlab.NewSystemFailureError(err)
	}
	defer sshClient.Close()

	if err := sshutil.RunScript(sshClient, args[0], cfg.Shell); err != nil {
		var sshExitError *ssh.ExitError
		if errors.As(err, &sshExitError) {
			sshutil.PropagateSSHExitError(err)
		}
		return err
	}
	return nil
}
