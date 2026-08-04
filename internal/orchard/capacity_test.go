package orchard

import (
	"testing"
	"time"

	v1 "github.com/cirruslabs/orchard/pkg/resource/v1"
	"github.com/stretchr/testify/require"
)

func TestEvaluateCapacityAllowsWhenSlotsFree(t *testing.T) {
	cfg := DefaultConfig()
	workers := []v1.Worker{{
		Meta:      v1.Meta{Name: "mac-studio"},
		LastSeen:  time.Now(),
		Resources: v1.Resources{ResourceTartVMs: 2},
		Labels:    v1.Labels{},
	}}
	vms := []v1.VM{scheduledVM("gitlab-1", "mac-studio")}

	snapshot := EvaluateCapacity(workers, vms, cfg, 5*time.Minute)
	require.True(t, snapshot.CanSchedule, snapshot.Reason)
	require.Equal(t, uint64(1), snapshot.FreeSlots)
}

func TestEvaluateCapacityBlocksWhenFull(t *testing.T) {
	cfg := DefaultConfig()
	workers := []v1.Worker{{
		Meta:      v1.Meta{Name: "mac-studio"},
		LastSeen:  time.Now(),
		Resources: v1.Resources{ResourceTartVMs: 2},
	}}
	vms := []v1.VM{
		scheduledVM("gitlab-a", "mac-studio"),
		scheduledVM("gitlab-b", "mac-studio"),
	}

	snapshot := EvaluateCapacity(workers, vms, cfg, 5*time.Minute)
	require.False(t, snapshot.CanSchedule, snapshot.Reason)
}

func TestEvaluateCapacityBlocksOnPendingPileup(t *testing.T) {
	cfg := DefaultConfig()
	workers := []v1.Worker{{
		Meta:      v1.Meta{Name: "mac-studio"},
		LastSeen:  time.Now(),
		Resources: v1.Resources{ResourceTartVMs: 2},
	}}
	vms := []v1.VM{
		scheduledVM("gitlab-a", "mac-studio"),
		{
			Meta:      v1.Meta{Name: "gitlab-pending"},
			Status:    v1.VMStatusPending,
			Resources: v1.Resources{ResourceTartVMs: 1},
		},
	}

	snapshot := EvaluateCapacity(workers, vms, cfg, 5*time.Minute)
	require.False(t, snapshot.CanSchedule, snapshot.Reason)
}

func TestEvaluateCapacityRespectsMaxConcurrent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrentVMs = 1
	workers := []v1.Worker{{
		Meta:      v1.Meta{Name: "mac-studio"},
		LastSeen:  time.Now(),
		Resources: v1.Resources{ResourceTartVMs: 2},
	}}
	vms := []v1.VM{scheduledVM("gitlab-a", "mac-studio")}

	snapshot := EvaluateCapacity(workers, vms, cfg, 5*time.Minute)
	require.False(t, snapshot.CanSchedule, snapshot.Reason)
}

func TestEvaluateCapacityRespectsLabels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Labels["model"] = "macstudio"
	workers := []v1.Worker{{
		Meta:      v1.Meta{Name: "mac-mini"},
		LastSeen:  time.Now(),
		Resources: v1.Resources{ResourceTartVMs: 2},
		Labels:    v1.Labels{"model": "macmini"},
	}}

	snapshot := EvaluateCapacity(workers, nil, cfg, 5*time.Minute)
	require.False(t, snapshot.CanSchedule, snapshot.Reason)
}

func scheduledVM(name, worker string) v1.VM {
	return v1.VM{
		Meta:      v1.Meta{Name: name},
		Status:    v1.VMStatusRunning,
		Worker:    worker,
		Resources: v1.Resources{ResourceTartVMs: 1},
	}
}
