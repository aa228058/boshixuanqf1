package task

import (
	"testing"

	"github.com/google/uuid"
)

func TestAutoSchedulerAppliesDefaultQuotaToUnknownTenant(t *testing.T) {
	scheduler := NewAutoScheduler(2800)
	defer scheduler.Stop()

	tenantID := uuid.New()
	if !scheduler.TryAcquire(tenantID) {
		t.Fatal("first acquisition should fit the default quota of one")
	}
	if scheduler.TryAcquire(tenantID) {
		t.Fatal("second acquisition must not exceed the default tenant quota")
	}
	if got := scheduler.GetTenantCurrent(tenantID); got != 1 {
		t.Fatalf("current usage = %d, want 1", got)
	}

	scheduler.Release(tenantID)
	if !scheduler.TryAcquire(tenantID) {
		t.Fatal("acquisition should succeed after release")
	}
}

func TestAutoSchedulerRegisterTenantUpdatesDefaultQuota(t *testing.T) {
	scheduler := NewAutoScheduler(2800)
	defer scheduler.Stop()

	tenantID := uuid.New()
	if !scheduler.TryAcquire(tenantID) {
		t.Fatal("first acquisition should succeed")
	}
	scheduler.RegisterTenant(tenantID, 2)
	if !scheduler.TryAcquire(tenantID) {
		t.Fatal("registered quota should allow a second concurrent task")
	}
	if got := scheduler.GetTenantCurrent(tenantID); got != 2 {
		t.Fatalf("current usage = %d, want 2", got)
	}
}
