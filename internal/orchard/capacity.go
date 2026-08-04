package orchard

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cirruslabs/orchard/pkg/client"
	v1 "github.com/cirruslabs/orchard/pkg/resource/v1"
)

// CapacitySnapshot describes whether a new VM with the given requirements can be scheduled.
type CapacitySnapshot struct {
	EligibleWorkers   int
	FreeSlots         uint64
	ManagedVMCount    uint64
	PendingManagedVMs uint64
	CanSchedule       bool
	Reason            string
}

// EvaluateCapacity inspects Orchard workers/VMs and decides if one more VM can be created.
//
// Orchard accepts Create even when no worker can fit the VM (it stays pending). This gate
// prevents that pile-up by requiring a free matching slot before Create.
func EvaluateCapacity(
	workers []v1.Worker,
	vms []v1.VM,
	cfg Config,
	offlineTimeout time.Duration,
) CapacitySnapshot {
	requested := cfg.Resources.Copy()
	if len(requested) == 0 {
		requested = v1.Resources{ResourceTartVMs: 1}
	}
	requestedSlots := slotCost(requested)

	usedByWorker := map[string]v1.Resources{}
	var managedCount uint64
	var pendingManaged uint64
	var pendingCompetingSlots uint64

	for _, vm := range vms {
		managed := IsManagedVMName(vm.Name)
		if managed {
			managedCount++
		}

		if vm.Status == v1.VMStatusFailed {
			continue
		}

		if vm.IsScheduled() {
			if usedByWorker[vm.Worker] == nil {
				usedByWorker[vm.Worker] = v1.Resources{}
			}
			usedByWorker[vm.Worker].Add(effectiveVMResources(vm))
			continue
		}

		// Pending VM
		if managed {
			pendingManaged++
		}
		if labelsCompatible(cfg.Labels, vm.Labels) {
			pendingCompetingSlots += slotCost(effectiveVMResources(vm))
		}
	}

	var freeSlots uint64
	eligible := 0

	for _, worker := range workers {
		if worker.Offline(offlineTimeout) || worker.SchedulingPaused {
			continue
		}
		if !worker.Labels.Contains(cfg.Labels) {
			continue
		}
		eligible++

		remaining := worker.Resources.Subtracted(usedByWorker[worker.Name])
		if !remaining.CanFit(requested) {
			continue
		}
		freeSlots += remainingSlots(remaining, requestedSlots)
	}

	snapshot := CapacitySnapshot{
		EligibleWorkers:   eligible,
		FreeSlots:         freeSlots,
		ManagedVMCount:    managedCount,
		PendingManagedVMs: pendingManaged,
	}

	if eligible == 0 {
		snapshot.CanSchedule = false
		snapshot.Reason = "no eligible online Orchard workers match the requested labels"
		return snapshot
	}

	if cfg.MaxConcurrentVMs > 0 && managedCount >= cfg.MaxConcurrentVMs {
		snapshot.CanSchedule = false
		snapshot.Reason = fmt.Sprintf(
			"managed VM limit reached (%d/%d)", managedCount, cfg.MaxConcurrentVMs,
		)
		return snapshot
	}

	if freeSlots <= pendingCompetingSlots {
		snapshot.CanSchedule = false
		snapshot.Reason = fmt.Sprintf(
			"no free VM slots (free=%d, pending_competing=%d, eligible_workers=%d)",
			freeSlots, pendingCompetingSlots, eligible,
		)
		return snapshot
	}

	snapshot.CanSchedule = true
	snapshot.Reason = fmt.Sprintf(
		"capacity available (free=%d, pending_competing=%d, managed=%d)",
		freeSlots, pendingCompetingSlots, managedCount,
	)
	return snapshot
}

func WaitForCapacity(ctx context.Context, c *client.Client, cfg Config) (CapacitySnapshot, error) {
	deadline := time.Now().Add(cfg.CapacityWaitTimeout)
	attempt := 0

	for {
		attempt++
		workers, err := c.Workers().List(ctx)
		if err != nil {
			return CapacitySnapshot{}, fmt.Errorf("list workers: %w", err)
		}
		vms, err := c.VMs().List(ctx)
		if err != nil {
			return CapacitySnapshot{}, fmt.Errorf("list vms: %w", err)
		}

		snapshot := EvaluateCapacity(workers, vms, cfg, cfg.WorkerOfflineTimeout)
		if snapshot.CanSchedule {
			log.Printf("Capacity gate: %s", snapshot.Reason)
			return snapshot, nil
		}

		if time.Now().After(deadline) {
			return snapshot, fmt.Errorf(
				"timed out waiting for Orchard capacity after %s: %s",
				cfg.CapacityWaitTimeout, snapshot.Reason,
			)
		}

		log.Printf("Capacity gate (attempt %d): %s; waiting %s...",
			attempt, snapshot.Reason, cfg.CapacityPollInterval)

		timer := time.NewTimer(cfg.CapacityPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return snapshot, ctx.Err()
		case <-timer.C:
		}
	}
}

func effectiveVMResources(vm v1.VM) v1.Resources {
	if len(vm.Resources) > 0 {
		return vm.Resources.Copy()
	}
	return v1.Resources{ResourceTartVMs: 1}
}

func slotCost(resources v1.Resources) uint64 {
	if resources[ResourceTartVMs] > 0 {
		return resources[ResourceTartVMs]
	}
	return 1
}

func remainingSlots(remaining v1.Resources, requestedSlots uint64) uint64 {
	if requestedSlots == 0 {
		requestedSlots = 1
	}
	if remaining[ResourceTartVMs] > 0 {
		return remaining[ResourceTartVMs] / requestedSlots
	}
	if remaining.CanFit(v1.Resources{ResourceTartVMs: requestedSlots}) {
		return 1
	}
	return 1
}

// labelsCompatible reports whether two label sets could schedule on overlapping workers.
func labelsCompatible(a, b v1.Labels) bool {
	return a.Contains(b) || b.Contains(a)
}
