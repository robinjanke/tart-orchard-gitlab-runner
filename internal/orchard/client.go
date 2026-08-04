package orchard

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/cirruslabs/orchard/pkg/client"
)

func NewClient(cfg Config) (*client.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := []client.Option{
		client.WithAddress(cfg.URL),
		client.WithCredentials(cfg.ServiceAccountName, cfg.ServiceAccountToken),
	}

	if cfg.TrustedCertificatePath != "" {
		cert, err := loadTrustedCertificate(cfg.TrustedCertificatePath)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.WithTrustedCertificate(cert))
	}

	c, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Orchard client: %w", err)
	}
	return c, nil
}

func loadTrustedCertificate(path string) (*x509.Certificate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Orchard trusted certificate %q: %w", path, err)
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("Orchard trusted certificate %q: no PEM block found", path)
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("Orchard trusted certificate %q: unexpected PEM type %q", path, block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse Orchard trusted certificate %q: %w", path, err)
	}
	return cert, nil
}
