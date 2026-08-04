package config

import (
	"encoding/json"
	"fmt"

	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/gitlab"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/version"
	"github.com/spf13/cobra"
)

var (
	buildsDir string
	cacheDir  string
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure GitLab Runner custom executor",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runConfig(); err != nil {
				return gitlab.NewSystemFailureError(err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&buildsDir, "builds-dir", "/Users/admin/builds",
		"builds directory inside the guest VM")
	cmd.Flags().StringVar(&cacheDir, "cache-dir", "/Users/admin/cache",
		"cache directory inside the guest VM")

	return cmd
}

func runConfig() error {
	out := struct {
		BuildsDir string            `json:"builds_dir"`
		CacheDir  string            `json:"cache_dir"`
		Driver    map[string]string `json:"driver"`
	}{
		BuildsDir: buildsDir,
		CacheDir:  cacheDir,
		Driver: map[string]string{
			"name":    "gitlab-orchard-executor",
			"version": version.FullVersion(),
		},
	}

	jsonBytes, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonBytes))
	return nil
}
