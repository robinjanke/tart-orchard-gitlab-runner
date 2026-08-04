package flags

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "github.com/cirruslabs/orchard/pkg/resource/v1"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/orchard"
	"github.com/spf13/cobra"
)

// OrchardFlags holds shared Orchard connection / VM flags.
type OrchardFlags struct {
	URL                    string
	ServiceAccountName     string
	ServiceAccountToken    string
	TrustedCertificatePath string
	CPU                    uint64
	Memory                 uint64
	DiskSize               uint64
	Labels                 []string
	Resources              []string
	ImagePullPolicy        string
	SSHUsername            string
	SSHPassword            string
	SSHPort                uint16
	Shell                  string
	Headless               bool
	Nested                 bool
	DefaultImage           string
	MaxConcurrentVMs       uint64
	CapacityWaitTimeout    time.Duration
	VMReadyTimeout         time.Duration
}

func (f *OrchardFlags) RegisterConnection(cmd *cobra.Command) {
	defaults := orchard.DefaultConfig()
	cmd.Flags().StringVar(&f.URL, "orchard-url", defaults.URL, "Orchard controller URL")
	cmd.Flags().StringVar(&f.ServiceAccountName, "orchard-service-account-name", defaults.ServiceAccountName, "Orchard service account name")
	cmd.Flags().StringVar(&f.ServiceAccountToken, "orchard-service-account-token", defaults.ServiceAccountToken, "Orchard service account token")
	cmd.Flags().StringVar(&f.TrustedCertificatePath, "orchard-trusted-certificate", defaults.TrustedCertificatePath, "PEM file with the Orchard controller TLS certificate (self-signed)")
}

func (f *OrchardFlags) RegisterVM(cmd *cobra.Command) {
	defaults := orchard.DefaultConfig()
	cmd.Flags().Uint64Var(&f.CPU, "cpu", 0, "VM CPU count override (0 = Orchard/worker default)")
	cmd.Flags().Uint64Var(&f.Memory, "memory", 0, "VM memory in MiB (0 = Orchard/worker default)")
	cmd.Flags().Uint64Var(&f.DiskSize, "disk-size", 0, "VM disk size in GB (0 = image default)")
	cmd.Flags().StringArrayVar(&f.Labels, "label", nil, "Orchard scheduling label key=value (repeatable)")
	cmd.Flags().StringArrayVar(&f.Resources, "resource", nil, "Orchard resource requirement key=value (repeatable)")
	cmd.Flags().StringVar(&f.ImagePullPolicy, "image-pull-policy", string(defaults.ImagePullPolicy), "IfNotPresent or Always")
	cmd.Flags().StringVar(&f.SSHUsername, "ssh-username", defaults.SSHUsername, "SSH username inside the VM")
	cmd.Flags().StringVar(&f.SSHPassword, "ssh-password", defaults.SSHPassword, "SSH password inside the VM")
	cmd.Flags().Uint16Var(&f.SSHPort, "ssh-port", defaults.SSHPort, "SSH port inside the VM")
	cmd.Flags().StringVar(&f.Shell, "shell", defaults.Shell, "Shell used to run GitLab scripts (default: system shell)")
	cmd.Flags().BoolVar(&f.Headless, "headless", defaults.Headless, "Run VM headless")
	cmd.Flags().BoolVar(&f.Nested, "nested", defaults.Nested, "Enable nested virtualization")
	cmd.Flags().StringVar(&f.DefaultImage, "default-image", defaults.DefaultImage, "Fallback image when the job does not set image:")
	cmd.Flags().Uint64Var(&f.MaxConcurrentVMs, "max-concurrent-vms", defaults.MaxConcurrentVMs, "Hard limit for VMs managed by this executor (0 = unlimited beyond cluster capacity)")
	cmd.Flags().DurationVar(&f.CapacityWaitTimeout, "capacity-wait-timeout", defaults.CapacityWaitTimeout, "How long prepare waits for a free Orchard VM slot")
	cmd.Flags().DurationVar(&f.VMReadyTimeout, "vm-ready-timeout", defaults.VMReadyTimeout, "How long prepare waits for the VM to become running")
}

func (f *OrchardFlags) Config() (orchard.Config, error) {
	cfg := orchard.DefaultConfig()
	if f.URL != "" {
		cfg.URL = f.URL
	}
	if f.ServiceAccountName != "" {
		cfg.ServiceAccountName = f.ServiceAccountName
	}
	if f.ServiceAccountToken != "" {
		cfg.ServiceAccountToken = f.ServiceAccountToken
	}
	if f.TrustedCertificatePath != "" {
		cfg.TrustedCertificatePath = f.TrustedCertificatePath
	}
	cfg.CPU = f.CPU
	cfg.Memory = f.Memory
	cfg.DiskSize = f.DiskSize
	cfg.SSHUsername = f.SSHUsername
	cfg.SSHPassword = f.SSHPassword
	cfg.SSHPort = f.SSHPort
	cfg.Shell = f.Shell
	cfg.Headless = f.Headless
	cfg.Nested = f.Nested
	cfg.DefaultImage = f.DefaultImage
	cfg.MaxConcurrentVMs = f.MaxConcurrentVMs
	cfg.CapacityWaitTimeout = f.CapacityWaitTimeout
	cfg.VMReadyTimeout = f.VMReadyTimeout

	if f.ImagePullPolicy != "" {
		policy, err := v1.NewImagePullPolicyFromString(f.ImagePullPolicy)
		if err != nil {
			return cfg, err
		}
		cfg.ImagePullPolicy = policy
	}

	for _, label := range f.Labels {
		key, value, ok := strings.Cut(label, "=")
		if !ok || key == "" {
			return cfg, fmt.Errorf("invalid --label %q (expected key=value)", label)
		}
		cfg.Labels[key] = value
	}

	for _, resource := range f.Resources {
		key, valueRaw, ok := strings.Cut(resource, "=")
		if !ok || key == "" {
			return cfg, fmt.Errorf("invalid --resource %q (expected key=value)", resource)
		}
		value, err := strconv.ParseUint(valueRaw, 10, 64)
		if err != nil {
			return cfg, fmt.Errorf("invalid --resource %q: %w", resource, err)
		}
		cfg.Resources[key] = value
	}

	if err := cfg.ApplyEnvOverrides(); err != nil {
		return cfg, err
	}
	return cfg, nil
}
