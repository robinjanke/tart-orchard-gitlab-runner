package prepare

import (
	"fmt"
	"log"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/flags"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/gitlab"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/orchard"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/sshutil"
	"github.com/spf13/cobra"
)

var (
	orchardFlags         flags.OrchardFlags
	allowedImagePatterns []string
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Create an Orchard VM for the GitLab job",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runPrepare(cmd); err != nil {
				return gitlab.NewSystemFailureError(err)
			}
			return nil
		},
	}

	orchardFlags.RegisterConnection(cmd)
	orchardFlags.RegisterVM(cmd)
	cmd.Flags().StringArrayVar(&allowedImagePatterns, "allow-image", nil,
		"only allow images matching this doublestar pattern (repeatable)")

	return cmd
}

func runPrepare(cmd *cobra.Command) error {
	cfg, err := orchardFlags.Config()
	if err != nil {
		return err
	}

	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return err
	}

	image := gitLabEnv.JobImage
	if image == "" {
		if cfg.DefaultImage == "" {
			return fmt.Errorf("job has no image and --default-image is not set")
		}
		image = cfg.DefaultImage
		log.Printf("No image provided, falling back to default: %s", image)
	}

	if err := ensureImageIsAllowed(image); err != nil {
		return err
	}

	client, err := orchard.NewClient(cfg)
	if err != nil {
		return err
	}

	if _, err := orchard.WaitForCapacity(cmd.Context(), client, cfg); err != nil {
		return err
	}

	vmSpec := orchard.BuildVM(gitLabEnv.VirtualMachineID(), image, cfg)
	if _, err := orchard.CreateAndWaitRunning(cmd.Context(), client, vmSpec, cfg.VMReadyTimeout); err != nil {
		return err
	}

	log.Println("Waiting for SSH via Orchard port-forward...")
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
		_ = orchard.DeleteVM(cmd.Context(), client, gitLabEnv.VirtualMachineID())
		return err
	}
	_ = sshClient.Close()

	log.Println("VM is ready.")
	return nil
}

func ensureImageIsAllowed(image string) error {
	if len(allowedImagePatterns) == 0 {
		return nil
	}
	for _, pattern := range allowedImagePatterns {
		match, err := doublestar.Match(pattern, image)
		if err != nil {
			return err
		}
		if match {
			return nil
		}
	}
	return fmt.Errorf("image %q is disallowed by GitLab Runner configuration", image)
}
