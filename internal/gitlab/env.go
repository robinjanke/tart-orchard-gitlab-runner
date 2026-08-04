package gitlab

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

var ErrGitLabEnv = errors.New("GitLab environment error")

type Env struct {
	JobID    string
	JobImage string
	Registry *Registry
}

type Registry struct {
	Address  string
	User     string
	Password string
}

func (e Env) VirtualMachineID() string {
	return fmt.Sprintf("gitlab-%s", e.JobID)
}

func InitEnv() (*Env, error) {
	result := &Env{}

	jobID, ok := os.LookupEnv("CUSTOM_ENV_CI_JOB_ID")
	if !ok {
		return nil, fmt.Errorf("%w: CUSTOM_ENV_CI_JOB_ID is missing", ErrGitLabEnv)
	}
	result.JobID = jobID
	result.JobImage = os.Getenv("CUSTOM_ENV_CI_JOB_IMAGE")

	ciRegistry, ciRegistryOK := os.LookupEnv("CUSTOM_ENV_CI_REGISTRY")
	ciRegistryUser, ciRegistryUserOK := os.LookupEnv("CUSTOM_ENV_CI_REGISTRY_USER")
	ciRegistryPassword, ciRegistryPasswordOK := os.LookupEnv("CUSTOM_ENV_CI_REGISTRY_PASSWORD")
	if ciRegistryOK && ciRegistryUserOK && ciRegistryPasswordOK {
		result.Registry = &Registry{
			Address:  ciRegistry,
			User:     ciRegistryUser,
			Password: ciRegistryPassword,
		}
	}

	return result, nil
}

func ParseExitCode(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	code, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return code
}
