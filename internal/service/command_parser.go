package service

import (
	"context"
	"fmt"
	"homelab-telemetry-api/internal/domain"
	"strings"
	"time"
)

type CommandParser struct {
	repo domain.SnapshotRepository
}

func NewCommandParser(repo domain.SnapshotRepository) *CommandParser {
	return &CommandParser{repo: repo}
}

func (p *CommandParser) Execute(ctx context.Context, req domain.TerminalCommandRequest) (*domain.TerminalCommandResponse, error) {
	cmd := strings.ToLower(strings.TrimSpace(req.Command))

	resp := &domain.TerminalCommandResponse{
		Command:   cmd,
		Timestamp: time.Now().UTC(),
	}

	switch cmd {
	case "help":
		resp.OutputText = "Available commands:\n" +
			"  status    - View cluster overview and resource usage\n" +
			"  services  - List monitored applications and health status\n" +
			"  nodes     - Inspect infrastructure nodes and hardware specs\n" +
			"  clear     - Clear terminal buffer\n" +
			"  help      - Show this help menu"
		resp.Data = map[string]interface{}{
			"commands": []string{"status", "services", "nodes", "clear", "help"},
		}

	case "status":
		overview, err := p.repo.GetOverView(ctx)
		if err != nil {
			resp.OutputText = "Error: Cluster telemetry snapshot not ready yet."
			return resp, nil
		}
		resp.OutputText = fmt.Sprintf(
			"Cluster: %s (%s)\nUptime: %ds | Nodes: %d/%d Healthy | Services: %d/%d Healthy\nCPU Usage: %.1f%% | Memory Usage: %.1f%%",
			overview.ClusterName, overview.Environment,
			overview.UptimeSeconds,
			overview.HealthyNodes, overview.TotalNodes,
			overview.HealthyServices, overview.TotalServices,
			overview.Resources.CPUUsagePercent, overview.Resources.MemoryUsagePercent,
		)
		resp.Data = overview

	case "services":
		services, err := p.repo.GetServices(ctx)
		if err != nil || len(services) == 0 {
			resp.OutputText = "No services monitored or snapshot not ready."
			return resp, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%-20s %-10s %-10s %s\n", "NAME", "STATUS", "LATENCY", "LAST CHECK"))
		for _, s := range services {
			sb.WriteString(fmt.Sprintf("%-20s %-10s %-10dms %s\n", s.Name, s.Status, s.LatencyMs, s.LastCheck.Format("15:04:05")))
		}
		resp.OutputText = strings.TrimSpace(sb.String())
		resp.Data = services

	case "nodes":
		nodes, err := p.repo.GetNodes(ctx)
		if err != nil || len(nodes) == 0 {
			resp.OutputText = "No nodes registered or snapshot not ready."
			return resp, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%-15s %-15s %-10s %-10s %s\n", "ID", "ROLE", "STATUS", "CPU%", "RAM (USED/TOT)"))
		for _, n := range nodes {
			sb.WriteString(fmt.Sprintf("%-15s %-15s %-10s %-10.1f %dMB/%dMB\n", n.ID, n.Role, n.Status, n.CPUPercent, n.MemoryUsedMB, n.MemoryTotalMB))
		}
		resp.OutputText = strings.TrimSpace(sb.String())
		resp.Data = nodes

	case "clear":
		resp.OutputText = ""

	default:
		resp.OutputText = fmt.Sprintf("Command not found: '%s'. Type 'help' for available commands.", cmd)
		return resp, domain.ErrCommandNotFound
	}

	return resp, nil
}