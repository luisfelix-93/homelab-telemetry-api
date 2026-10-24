package collector

import (
	"context"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"homelab-telemetry-api/internal/domain"
)

const bytesToMegabytes = 1024 * 1024

// SystemMetricsProvider abstrai as chamadas de sistema operacional para facilitar testes unitários.
type SystemMetricsProvider interface {
	CPUInfo(ctx context.Context) (cores int, usagePercent float64, err error)
	MemoryInfo(ctx context.Context) (totalBytes uint64, usedBytes uint64, err error)
	Uptime(ctx context.Context) (uptimeSeconds uint64, err error)
}

// gopsutilProvider é a implementação real que lê os dados do host local.
type gopsutilProvider struct{}

func (g *gopsutilProvider) CPUInfo(ctx context.Context) (int, float64, error) {
	cores, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		return 0, 0, err
	}

	// Intervalo 0 calcula a porcentagem desde a última chamada sem bloquear a goroutine
	percentages, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return 0, 0, err
	}

	var usage float64
	if len(percentages) > 0 {
		usage = percentages[0]
	}

	return cores, usage, nil
}

func (g *gopsutilProvider) MemoryInfo(ctx context.Context) (uint64, uint64, error) {
	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, 0, err
	}
	return vmStat.Total, vmStat.Used, nil
}

func (g *gopsutilProvider) Uptime(ctx context.Context) (uint64, error) {
	return host.UptimeWithContext(ctx)
}

// SystemNodeCollector implementa domain.NodeCollector.
type SystemNodeCollector struct {
	name     string
	nodeID   string
	nodeRole string
	provider SystemMetricsProvider
}

// Compile-time check garantindo a implementação da interface do domínio.
var _ domain.NodeCollector = (*SystemNodeCollector)(nil)

// NewSystemNodeCollector cria o coletor usando o gopsutil real para produção.
func NewSystemNodeCollector(name, nodeID, nodeRole string) *SystemNodeCollector {
	return &SystemNodeCollector{
		name:     name,
		nodeID:   nodeID,
		nodeRole: nodeRole,
		provider: &gopsutilProvider{},
	}
}

// NewSystemNodeCollectorWithProvider permite injetar mocks/fakes em testes unitários.
func NewSystemNodeCollectorWithProvider(name, nodeID, nodeRole string, provider SystemMetricsProvider) *SystemNodeCollector {
	return &SystemNodeCollector{
		name:     name,
		nodeID:   nodeID,
		nodeRole: nodeRole,
		provider: provider,
	}
}

func (c *SystemNodeCollector) Name() string {
	return c.name
}

func (c *SystemNodeCollector) CollectNodes(ctx context.Context) ([]domain.NodeTelemetry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cores, cpuPercent, err := c.provider.CPUInfo(ctx)
	if err != nil {
		return nil, err
	}

	memTotal, memUsed, err := c.provider.MemoryInfo(ctx)
	if err != nil {
		return nil, err
	}

	uptime, err := c.provider.Uptime(ctx)
	if err != nil {
		return nil, err
	}

	node := domain.NodeTelemetry{
		ID:            c.nodeID,
		Role:          c.nodeRole,
		Status:        "Ready",
		CPUCores:      cores,
		CPUPercent:    cpuPercent,
		MemoryTotalMB: int64(memTotal / bytesToMegabytes),
		MemoryUsedMB:  int64(memUsed / bytesToMegabytes),
		UptimeSeconds: uptime,
	}

	return []domain.NodeTelemetry{node}, nil
}