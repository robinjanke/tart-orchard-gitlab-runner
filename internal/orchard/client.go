package orchard

import (
	"fmt"

	"github.com/cirruslabs/orchard/pkg/client"
)

func NewClient(cfg Config) (*client.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	c, err := client.New(
		client.WithAddress(cfg.URL),
		client.WithCredentials(cfg.ServiceAccountName, cfg.ServiceAccountToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Orchard client: %w", err)
	}
	return c, nil
}
