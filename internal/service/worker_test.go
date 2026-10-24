package service_test

import (
	"context"
	"testing"
	

	"homelab-telemetry-api/internal/domain"
	"homelab-telemetry-api/internal/infra/cache"
	"homelab-telemetry-api/internal/service"
)

// mockServiceCollector simula um coletor de aplicações.
type mockServiceCollector struct {
	services []domain.ServiceTelemetry
	err      error
}

func (m *mockServiceCollector) Name() string { return "mock-service-collector" }
func (m *mockServiceCollector) CollectServices(ctx context.Context) ([]domain.ServiceTelemetry, error) {
	return m.services, m.err
}

// mockNodeCollector simula um coletor de nós do sistema.
type mockNodeCollector struct {
	nodes []domain.NodeTelemetry
	err   error
}

func (m *mockNodeCollector) Name() string { return "mock-node-collector" }
func (m *mockNodeCollector) CollectNodes(ctx context.Context) ([]domain.NodeTelemetry, error) {
	return m.nodes, m.err
}

func TestTelemetryWorker_RunOnce(t *testing.T) {
	ctx := context.Background()
	repo := cache.NewMemorySnapshotRepository()

	svcCollector := &mockServiceCollector{
		services: []domain.ServiceTelemetry{
			{Name: "api", Status: domain.StatusHealthy, LatencyMs: 10},
			{Name: "db", Status: domain.StatusDegraded, LatencyMs: 150},
		},
	}

	nodeCollector := &mockNodeCollector{
		nodes: []domain.NodeTelemetry{
			{ID: "node-01", Status: "Ready", CPUPercent: 20.0, MemoryTotalMB: 16384, MemoryUsedMB: 4096, UptimeSeconds: 1000},
		},
	}

	worker := service.NewTelemetryWorker(
		repo,
		[]domain.ServiceCollector{svcCollector},
		[]domain.NodeCollector{nodeCollector},
		"homelab-cluster",
		"production",
	)

	// Executa uma rodada síncrona
	err := worker.RunOnce(ctx)
	if err != nil {
		t.Fatalf("unexpected error running worker once: %v", err)
	}

	// Valida se os serviços foram salvos corretamente
	savedServices, err := repo.GetServices(ctx)
	if err != nil || len(savedServices) != 2 {
		t.Fatalf("expected 2 services in repo, got %d (err: %v)", len(savedServices), err)
	}

	// Valida se os nós foram salvos corretamente
	savedNodes, err := repo.GetNodes(ctx)
	if err != nil || len(savedNodes) != 1 {
		t.Fatalf("expected 1 node in repo, got %d (err: %v)", len(savedNodes), err)
	}

	// Valida se o ClusterOverview consolidado foi calculado e salvo corretamente
	overview, err := repo.GetOverView(ctx)
	if err != nil {
		t.Fatalf("expected overview to be present, got error: %v", err)
	}

	if overview.ClusterName != "homelab-cluster" {
		t.Errorf("expected cluster name 'homelab-cluster', got '%s'", overview.ClusterName)
	}
	if overview.Environment != "production" {
		t.Errorf("expected environment 'production', got '%s'", overview.Environment)
	}
	if overview.TotalServices != 2 || overview.HealthyServices != 1 {
		// Apenas 'api' está UP (Healthy). 'db' está DEGRADED.
		t.Errorf("expected 2 total and 1 healthy service, got total=%d healthy=%d", overview.TotalServices, overview.HealthyServices)
	}
	if overview.TotalNodes != 1 || overview.HealthyNodes != 1 {
		t.Errorf("expected 1 total and 1 healthy node, got total=%d healthy=%d", overview.TotalNodes, overview.HealthyNodes)
	}
	if overview.Resources.CPUUsagePercent != 20.0 {
		t.Errorf("expected 20.0 CPU usage percent, got %f", overview.Resources.CPUUsagePercent)
	}
}