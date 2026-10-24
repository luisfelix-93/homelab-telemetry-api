package service

import (
	"context"
	"log"
	"time"

	"homelab-telemetry-api/internal/domain"
)

type TelemetryWorker struct {
	repo            domain.SnapshotRepository
	serviceCollectors []domain.ServiceCollector
	nodeCollectors  []domain.NodeCollector
	clusterName     string
	environment     string
}

func NewTelemetryWorker(
	repo domain.SnapshotRepository,
	serviceCollectors []domain.ServiceCollector,
	nodeCollectors []domain.NodeCollector,
	clusterName string,
	environment string,
) *TelemetryWorker {
	return &TelemetryWorker{
		repo:              repo,
		serviceCollectors: serviceCollectors,
		nodeCollectors:    nodeCollectors,
		clusterName:       clusterName,
		environment:       environment,
	}
}

// Start inicia o loop assíncrono de coleta periódica.
func (w *TelemetryWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Executa uma coleta inicial imediatamente ao subir a aplicação
	if err := w.RunOnce(ctx); err != nil {
		log.Printf("[Worker] error on initial collection tick: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("[Worker] shutting down background collector...")
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil {
				log.Printf("[Worker] error during collection tick: %v", err)
			}
		}
	}
}

// RunOnce executa um ciclo completo de coleta, agregação e salvamento no cache.
func (w *TelemetryWorker) RunOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// 1. Coleta serviços de todos os ServiceCollectors configurados
	var allServices []domain.ServiceTelemetry
	for _, sc := range w.serviceCollectors {
		services, err := sc.CollectServices(ctx)
		if err != nil {
			log.Printf("[Worker] collector %s failed: %v", sc.Name(), err)
			continue
		}
		allServices = append(allServices, services...)
	}

	// 2. Coleta nós de todos os NodeCollectors configurados
	var allNodes []domain.NodeTelemetry
	for _, nc := range w.nodeCollectors {
		nodes, err := nc.CollectNodes(ctx)
		if err != nil {
			log.Printf("[Worker] collector %s failed: %v", nc.Name(), err)
			continue
		}
		allNodes = append(allNodes, nodes...)
	}

	// 3. Salva raw services e nodes no repositório
	if err := w.repo.SaveServices(ctx, allServices); err != nil {
		return err
	}
	if err := w.repo.SaveNodes(ctx, allNodes); err != nil {
		return err
	}

	// 4. Calcula métricas consolidadas para o ClusterOverview
	overview := w.buildOverview(allServices, allNodes)
	if err := w.repo.SaveOverView(ctx, overview); err != nil {
		return err
	}

	return nil
}

func (w *TelemetryWorker) buildOverview(services []domain.ServiceTelemetry, nodes []domain.NodeTelemetry) *domain.ClusterOverview {
	totalServices := len(services)
	healthyServices := 0
	for _, s := range services {
		if s.Status == domain.StatusHealthy {
			healthyServices++
		}
	}

	totalNodes := len(nodes)
	healthyNodes := 0
	var totalCPUPercent float64
	var totalMemUsedBytes, totalMemTotalBytes uint64
	var maxUptime uint64

	for _, n := range nodes {
		if n.Status == "Ready" {
			healthyNodes++
		}
		totalCPUPercent += n.CPUPercent
		totalMemUsedBytes += uint64(n.MemoryUsedMB) * 1024 * 1024
		totalMemTotalBytes += uint64(n.MemoryTotalMB) * 1024 * 1024
		if n.UptimeSeconds > maxUptime {
			maxUptime = n.UptimeSeconds
		}
	}

	var avgCPUPercent float64
	if totalNodes > 0 {
		avgCPUPercent = totalCPUPercent / float64(totalNodes)
	}

	var memUsagePercent float64
	if totalMemTotalBytes > 0 {
		memUsagePercent = (float64(totalMemUsedBytes) / float64(totalMemTotalBytes)) * 100.0
	}

	return &domain.ClusterOverview{
		ClusterName:   w.clusterName,
		Environment:   w.environment,
		UptimeSeconds: maxUptime,
		LastUpdated:   time.Now().UTC(),
		Resources: domain.ClusterResources{
			CPUUsagePercent:    avgCPUPercent,
			MemoryUsedBytes:    totalMemUsedBytes,
			MemoryTotalBytes:   totalMemTotalBytes,
			MemoryUsagePercent: float32(memUsagePercent),
		},
		TotalNodes:      totalNodes,
		HealthyNodes:    healthyNodes,
		TotalServices:   totalServices,
		HealthyServices: healthyServices,
	}
}