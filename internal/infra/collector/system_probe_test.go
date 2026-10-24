package collector_test

import (
	"context"
	"testing"

	"homelab-telemetry-api/internal/infra/collector"
)

type fakeMetricsProvider struct {
	cpuCores   int
	cpuPercent float64
	memTotal   uint64
	memUsed    uint64
	uptime     uint64
	err        error
}

func (f *fakeMetricsProvider) CPUInfo(ctx context.Context) (int, float64, error) {
	if f.err != nil {
		return 0, 0, f.err
	}
	return f.cpuCores, f.cpuPercent, nil
}

func (f *fakeMetricsProvider) MemoryInfo(ctx context.Context) (uint64, uint64, error) {
	if f.err != nil {
		return 0, 0, f.err
	}
	return f.memTotal, f.memUsed, nil
}

func (f *fakeMetricsProvider) Uptime(ctx context.Context) (uint64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.uptime, nil
}

func TestSystemNodeCollector_CollectNodes(t *testing.T) {
	t.Run("should collect local node metrics and convert memory to MB accurately", func(t *testing.T) {
		fakeProvider := &fakeMetricsProvider{
			cpuCores:   4,
			cpuPercent: 25.5,
			memTotal:   8589934592, // 8192 MB
			memUsed:    4294967296, // 4096 MB
			uptime:     3600,
		}

		probe := collector.NewSystemNodeCollectorWithProvider(
			"homelab-node-collector",
			"node-homelab-01",
			"control-plane",
			fakeProvider,
		)

		nodes, err := probe.CollectNodes(context.Background())
		if err != nil {
			t.Fatalf("unexpected error during collection: %v", err)
		}

		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}

		node := nodes[0]
		if node.ID != "node-homelab-01" {
			t.Errorf("expected node ID 'node-homelab-01', got %s", node.ID)
		}
		if node.Role != "control-plane" {
			t.Errorf("expected node Role 'control-plane', got %s", node.Role)
		}
		if node.Status != "Ready" {
			t.Errorf("expected node status 'Ready', got %s", node.Status)
		}
		if node.CPUCores != 4 {
			t.Errorf("expected 4 CPU cores, got %d", node.CPUCores)
		}
		if node.CPUPercent != 25.5 {
			t.Errorf("expected 25.5%% CPU, got %f", node.CPUPercent)
		}
		if node.MemoryTotalMB != 8192 {
			t.Errorf("expected 8192 MB total, got %d", node.MemoryTotalMB)
		}
		if node.MemoryUsedMB != 4096 {
			t.Errorf("expected 4096 MB used, got %d", node.MemoryUsedMB)
		}
		if node.UptimeSeconds != 3600 {
			t.Errorf("expected 3600s uptime, got %d", node.UptimeSeconds)
		}
	})
}