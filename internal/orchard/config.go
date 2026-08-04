package orchard

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	v1 "github.com/cirruslabs/orchard/pkg/resource/v1"
)

const (
	// VMNamePrefix identifies VMs created by this executor (GitLab job VMs).
	VMNamePrefix    = "gitlab-"
	ResourceTartVMs = v1.ResourceTartVMs
)

type Config struct {
	URL                     string
	ServiceAccountName      string
	ServiceAccountToken     string
	TrustedCertificatePath  string
	CPU                     uint64
	Memory                  uint64
	DiskSize                uint64
	Labels                  v1.Labels
	Resources               v1.Resources
	ImagePullPolicy         v1.ImagePullPolicy
	SSHUsername             string
	SSHPassword             string
	SSHPort                 uint16
	Shell                   string
	Headless                bool
	Nested                  bool
	DefaultImage            string
	MaxConcurrentVMs        uint64
	CapacityWaitTimeout     time.Duration
	CapacityPollInterval    time.Duration
	VMReadyTimeout          time.Duration
	PortForwardWait         uint16
	WorkerOfflineTimeout    time.Duration
}

func DefaultConfig() Config {
	return Config{
		URL:                    firstEnv("ORCHARD_URL", "ORCHARD_EXECUTOR_URL"),
		ServiceAccountName:     firstEnv("ORCHARD_SERVICE_ACCOUNT_NAME", "ORCHARD_EXECUTOR_SERVICE_ACCOUNT_NAME"),
		ServiceAccountToken:    firstEnv("ORCHARD_SERVICE_ACCOUNT_TOKEN", "ORCHARD_EXECUTOR_SERVICE_ACCOUNT_TOKEN"),
		TrustedCertificatePath: firstEnv("ORCHARD_TRUSTED_CERTIFICATE", "ORCHARD_EXECUTOR_TRUSTED_CERTIFICATE"),
		SSHUsername:            envOr("ORCHARD_EXECUTOR_SSH_USERNAME", "admin"),
		SSHPassword:            envOr("ORCHARD_EXECUTOR_SSH_PASSWORD", "admin"),
		SSHPort:                22,
		Shell:                os.Getenv("ORCHARD_EXECUTOR_SHELL"),
		Headless:             envBool("ORCHARD_EXECUTOR_HEADLESS", true),
		Nested:               envBool("ORCHARD_EXECUTOR_NESTED", false),
		DefaultImage:         os.Getenv("ORCHARD_EXECUTOR_DEFAULT_IMAGE"),
		MaxConcurrentVMs:     0,
		CapacityWaitTimeout:  10 * time.Minute,
		CapacityPollInterval: 5 * time.Second,
		VMReadyTimeout:       15 * time.Minute,
		PortForwardWait:      120,
		WorkerOfflineTimeout: 5 * time.Minute,
		ImagePullPolicy: v1.ImagePullPolicyIfNotPresent,
		Labels:          v1.Labels{},
		Resources: v1.Resources{
			ResourceTartVMs: 1,
		},
	}
}

func IsManagedVMName(name string) bool {
	return strings.HasPrefix(name, VMNamePrefix)
}

func (c *Config) ApplyEnvOverrides() error {
	if raw := firstEnv("ORCHARD_URL", "ORCHARD_EXECUTOR_URL"); raw != "" {
		c.URL = raw
	}
	if raw := firstEnv("ORCHARD_SERVICE_ACCOUNT_NAME", "ORCHARD_EXECUTOR_SERVICE_ACCOUNT_NAME"); raw != "" {
		c.ServiceAccountName = raw
	}
	if raw := firstEnv("ORCHARD_SERVICE_ACCOUNT_TOKEN", "ORCHARD_EXECUTOR_SERVICE_ACCOUNT_TOKEN"); raw != "" {
		c.ServiceAccountToken = raw
	}
	if raw := firstEnv("ORCHARD_TRUSTED_CERTIFICATE", "ORCHARD_EXECUTOR_TRUSTED_CERTIFICATE"); raw != "" {
		c.TrustedCertificatePath = raw
	}
	if raw := os.Getenv("ORCHARD_EXECUTOR_CPU"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ORCHARD_EXECUTOR_CPU: %w", err)
		}
		c.CPU = v
	}
	if raw := os.Getenv("CUSTOM_ENV_ORCHARD_EXECUTOR_CPU"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid CUSTOM_ENV_ORCHARD_EXECUTOR_CPU: %w", err)
		}
		c.CPU = v
	}
	if raw := os.Getenv("ORCHARD_EXECUTOR_MEMORY"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ORCHARD_EXECUTOR_MEMORY: %w", err)
		}
		c.Memory = v
	}
	if raw := os.Getenv("CUSTOM_ENV_ORCHARD_EXECUTOR_MEMORY"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid CUSTOM_ENV_ORCHARD_EXECUTOR_MEMORY: %w", err)
		}
		c.Memory = v
	}
	if raw := os.Getenv("ORCHARD_EXECUTOR_DISK_SIZE"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ORCHARD_EXECUTOR_DISK_SIZE: %w", err)
		}
		c.DiskSize = v
	}
	if raw := firstEnv("ORCHARD_EXECUTOR_LABELS", "CUSTOM_ENV_ORCHARD_EXECUTOR_LABELS"); raw != "" {
		labels, err := parseKeyValueList(raw)
		if err != nil {
			return fmt.Errorf("invalid ORCHARD_EXECUTOR_LABELS: %w", err)
		}
		for k, v := range labels {
			c.Labels[k] = v
		}
	}
	if raw := firstEnv("ORCHARD_EXECUTOR_RESOURCES", "CUSTOM_ENV_ORCHARD_EXECUTOR_RESOURCES"); raw != "" {
		resources, err := parseUintKeyValueList(raw)
		if err != nil {
			return fmt.Errorf("invalid ORCHARD_EXECUTOR_RESOURCES: %w", err)
		}
		for k, v := range resources {
			c.Resources[k] = v
		}
	}
	if raw := os.Getenv("ORCHARD_EXECUTOR_IMAGE_PULL_POLICY"); raw != "" {
		policy, err := v1.NewImagePullPolicyFromString(raw)
		if err != nil {
			return err
		}
		c.ImagePullPolicy = policy
	}
	if raw := os.Getenv("ORCHARD_EXECUTOR_MAX_CONCURRENT_VMS"); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ORCHARD_EXECUTOR_MAX_CONCURRENT_VMS: %w", err)
		}
		c.MaxConcurrentVMs = v
	}
	if raw := os.Getenv("ORCHARD_EXECUTOR_CAPACITY_WAIT_TIMEOUT"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid ORCHARD_EXECUTOR_CAPACITY_WAIT_TIMEOUT: %w", err)
		}
		c.CapacityWaitTimeout = d
	}
	if user := os.Getenv("CUSTOM_ENV_ORCHARD_EXECUTOR_SSH_USERNAME"); user != "" {
		c.SSHUsername = user
	}
	if pass := os.Getenv("CUSTOM_ENV_ORCHARD_EXECUTOR_SSH_PASSWORD"); pass != "" {
		c.SSHPassword = pass
	}
	return nil
}

func (c Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("Orchard URL is required (set --orchard-url or ORCHARD_URL)")
	}
	if c.ServiceAccountName == "" || c.ServiceAccountToken == "" {
		return fmt.Errorf("Orchard service account credentials are required")
	}
	return nil
}

func parseKeyValueList(raw string) (map[string]string, error) {
	result := map[string]string{}
	for _, part := range splitCSV(raw) {
		key, value, ok := strings.Cut(part, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("expected key=value, got %q", part)
		}
		result[key] = value
	}
	return result, nil
}

func parseUintKeyValueList(raw string) (map[string]uint64, error) {
	result := map[string]uint64{}
	for _, part := range splitCSV(raw) {
		key, valueRaw, ok := strings.Cut(part, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("expected key=value, got %q", part)
		}
		value, err := strconv.ParseUint(valueRaw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %q: %w", key, err)
		}
		result[key] = value
	}
	return result, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
