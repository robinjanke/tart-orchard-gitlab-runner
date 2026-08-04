package orchard

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cirruslabs/orchard/pkg/client"
	v1 "github.com/cirruslabs/orchard/pkg/resource/v1"
)

func BuildVM(name, image string, cfg Config) *v1.VM {
	vm := &v1.VM{
		Meta: v1.Meta{
			Name: name,
		},
		Image:           image,
		ImagePullPolicy: cfg.ImagePullPolicy,
		CPU:             cfg.CPU,
		Memory:          cfg.Memory,
		DiskSize:        cfg.DiskSize,
		Headless:        cfg.Headless,
		Nested:          cfg.Nested,
		Username:        cfg.SSHUsername,
		Password:        cfg.SSHPassword,
		Resources:       cfg.Resources.Copy(),
		Labels:          copyLabels(cfg.Labels),
		RestartPolicy:   v1.RestartPolicyNever,
	}
	return vm
}

func CreateAndWaitRunning(ctx context.Context, c *client.Client, vm *v1.VM, readyTimeout time.Duration) (*v1.VM, error) {
	log.Printf("Creating Orchard VM %q from image %q...", vm.Name, vm.Image)
	if err := c.VMs().Create(ctx, vm); err != nil {
		return nil, fmt.Errorf("create VM: %w", err)
	}

	deadline := time.Now().Add(readyTimeout)
	for {
		current, err := c.VMs().Get(ctx, vm.Name)
		if err != nil {
			return nil, fmt.Errorf("get VM: %w", err)
		}

		switch current.Status {
		case v1.VMStatusRunning:
			log.Printf("VM %q is running on worker %q", current.Name, current.Worker)
			return current, nil
		case v1.VMStatusFailed:
			return current, fmt.Errorf("VM %q failed: %s", current.Name, current.StatusMessage)
		default:
			log.Printf("VM %q status=%s worker=%q — waiting...",
				current.Name, current.Status, current.Worker)
		}

		if time.Now().After(deadline) {
			_ = c.VMs().Delete(context.Background(), vm.Name)
			return current, fmt.Errorf("timed out waiting for VM %q to become running after %s (last status=%s)",
				vm.Name, readyTimeout, current.Status)
		}

		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = c.VMs().Delete(context.Background(), vm.Name)
			return current, ctx.Err()
		case <-timer.C:
		}
	}
}

func DeleteVM(ctx context.Context, c *client.Client, name string) error {
	log.Printf("Deleting Orchard VM %q...", name)
	if err := c.VMs().Delete(ctx, name); err != nil {
		return fmt.Errorf("delete VM %q: %w", name, err)
	}
	return nil
}

func copyLabels(in v1.Labels) v1.Labels {
	out := v1.Labels{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
