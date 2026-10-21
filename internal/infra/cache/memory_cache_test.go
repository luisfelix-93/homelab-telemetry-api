package cache_test

import (
	"context"
	"errors"
	"homelab-telemetry-api/internal/domain"
	"homelab-telemetry-api/internal/infra/cache"
	"testing"
	"time"
)

func TestMemorySnapshotRepository_Overview(t *testing.T) {
	ctx := context.Background()
	repo := cache.NewMemorySnapshotRepository()
	
	t.Run("should return ErrSnapshotNotReady when overview is not initialized", func(t *testing.T){
		overview, err := repo.GetOverView(ctx)
		if !errors.Is(err, domain.ErrSnapshotNotReady) {
			t.Fatalf("expected error %v, got %v", domain.ErrSnapshotNotReady, err)
		}
		if overview != nil {
			t.Fatalf("expected nil overview, got %v", overview)
		}
	})

	t.Run("Should save and retrieve overview with memory isolation", func(t *testing.T) {
		now := time.Now().UTC()
		input := &domain.ClusterOverview{
			ClusterName:     "homelab-k3s",
			Environment:     "production",
			UptimeSeconds:   3600,
			LastUpdated:     now,
			TotalNodes:      2,
			HealthyNodes:    2,
			TotalServices:   5,
			HealthyServices: 5,
		}
		
		if err := repo.SaveOverView(ctx, input); err != nil {
			t.Fatalf("unexpected error on save: %v", err)
		}

		// Mutando o objeto original para testar isolamento na entrada
		input.ClusterName = "mutated"
		input.HealthyNodes = 0

		retrieved, err := repo.GetOverView(ctx)
		if err != nil {
			t.Fatalf("unexpected error on get: %v", err)
		}

		if retrieved == nil {
			t.Fatalf("expected non-nil overview, got nil")
		}
		if retrieved == input {
			t.Fatalf("expected value semantic (no pointer aliasing), got same pointer")
		}

		if retrieved.ClusterName != "homelab-k3s" || retrieved.HealthyNodes != 2 || retrieved.HealthyServices != 5 {
			t.Fatalf("overview data corrupted after mutation")
		}

		if retrieved.LastUpdated != now {
			t.Fatalf("LastUpdated mismatch: expected %v, got %v", now, retrieved.LastUpdated)
		}

		// Mutando o objeto recuperado para garantir que não altera o cache interno
		retrieved.HealthyServices = 0
		retrievedAgain, _ := repo.GetOverView(ctx)
		if retrievedAgain.HealthyServices != 5 {
			t.Errorf("cache internal state was mutated externally! expected 5, got %d", retrievedAgain.HealthyServices)
		}
	})

	t.Run("GetNodes: must return ErrSnapshotNotReady before first save", func(t *testing.T) {
		nodes, err := repo.GetNodes(ctx)
		if !errors.Is(err, domain.ErrSnapshotNotReady) {
			t.Fatalf("expected error %v, got %v", domain.ErrSnapshotNotReady, err)
		}
		if nodes != nil {
			t.Fatalf("expected nil nodes, got %v", nodes)
		}
	})

	t.Run("Should save and retrieve nodes with memory isolation", func(t *testing.T) {
		inputNodes := []domain.NodeTelemetry{
			{
				ID:            "node-01",
				Role:          "control-plane",
				Status:        "Ready",
				CPUCores:      4,
				CPUPercent:    10.5,
				MemoryTotalMB: 8192,
				MemoryUsedMB:  4096,
			},
			{
				ID:            "node-02",
				Role:          "worker",
				Status:        "Ready",
				CPUCores:      8,
				CPUPercent:    25.0,
				MemoryTotalMB: 16384,
				MemoryUsedMB:  8192,
			},
		}
		
		if err := repo.SaveNodes(ctx, inputNodes); err != nil {
			t.Fatalf("unexpected error on save nodes: %v", err)
		}
		
		retrievedNodes, err := repo.GetNodes(ctx)
		if err != nil {
			t.Fatalf("unexpected error on get nodes: %v", err)
		}
		
		if len(retrievedNodes) != 2 || len(inputNodes) != len(retrievedNodes) {
			t.Fatalf("node count mismatch: expected %d, got %d", len(inputNodes), len(retrievedNodes))
		}
		
		// Verify isolation
		retrievedNodes[0].CPUPercent = 99.0
		retrievedNodes[0].Status = "NotReady"
		
		// We need to fetch again to check the internal state
		finalNodes, err := repo.GetNodes(ctx)
		if err != nil {
			t.Fatalf("unexpected error on get nodes (second call): %v", err)
		}
		
		// The first node's CPUPercent should still be 10.5, not 99.0
		if finalNodes[0].CPUPercent != 10.5 {
			t.Errorf("node isolation failed: CPUPercent changed from 10.5 to %f", finalNodes[0].CPUPercent)
		}
		
		// Verify the second node was copied correctly (check it's not the same pointer)
		if &retrievedNodes[1] == &finalNodes[1] {
			t.Errorf("node pointer aliasing detected!")
		}
	})

	t.Run("GetServices: must return ErrSnapshotNotReady before first save", func(t *testing.T) {
		services, err := repo.GetServices(ctx)
		if !errors.Is(err, domain.ErrSnapshotNotReady) {
			t.Fatalf("expected error %v, got %v", domain.ErrSnapshotNotReady, err)
		}
		if services != nil {
			t.Fatalf("expected nil services, got %v", services)
		}
	})

	t.Run("Should save and retrieve services with memory isolation", func(t *testing.T) {
		now := time.Now()
		inputServices := []domain.ServiceTelemetry{
			{
				Name:      "auth-service",
				Status:    domain.SatusHealthy,
				LatencyMs: 25,
				LastCheck: now,
				Message:   "OK",
			},
			{
				Name:      "db-service",
				Status:    domain.StatusDegraded,
				LatencyMs: 150,
				LastCheck: now,
				Message:   "Slow queries detected",
			},
		}

		if err := repo.SaveServices(ctx, inputServices); err != nil {
			t.Fatalf("unexpected error on save services: %v", err)
		}

		retrievedServices, err := repo.GetServices(ctx)
		if err != nil {
			t.Fatalf("unexpected error on get services: %v", err)
		}

		if len(retrievedServices) != 2 || len(inputServices) != len(retrievedServices) {
			t.Fatalf("service count mismatch: expected %d, got %d", len(inputServices), len(retrievedServices))
		}

		// Verify isolation
		retrievedServices[0].LatencyMs = 500
		retrievedServices[0].Status = domain.StatusDown

		finalServices, err := repo.GetServices(ctx)
		if err != nil {
			t.Fatalf("unexpected error on get services (second call): %v", err)
		}

		if finalServices[0].LatencyMs != 25 {
			t.Errorf("service isolation failed: LatencyMs changed from 25 to %d", finalServices[0].LatencyMs)
		}
	})
}