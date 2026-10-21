package domain

import (
	"errors"
	"time"
)

var (
	ErrCommandNotFound = errors.New("command not found")
	ErrEmptyCommand    = errors.New("empty command")
	ErrServiceNotFound = errors.New("service not found")
	ErrSnapshotNotReady = errors.New("telemetry snapshot not ready yet")
)

type ServiceStatus string

const (
	SatusHealthy   ServiceStatus = "UP"
	StatusDegraded ServiceStatus = "DEGRADED"
	StatusDown     ServiceStatus = "DOWN"
)

// ServiceTelemetry representa a saúde e métricas de um serviço monitorado
type ServiceTelemetry struct {
	Name      string        `json:"name"`
	Status    ServiceStatus `json:"status"`
	LatencyMs int64         `json:"latency_ms"`
	LastCheck time.Time     `json:"last_check"`
	Message   string        `json:"message"`
}

// NodeTelemetry traz o consumo e estado de uma máquina/host do homelab
type NodeTelemetry struct {
	ID            string  `json:"id"`
	Role          string  `json:"role"` // ex: "control-plane", "worker", "storage"
	Status        string  `json:"status"`
	CPUCores      int     `json:"cpu_cores"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryTotalMB int64   `json:"memory_total_mb"`
	MemoryUsedMB  int64   `json:"memory_used_mb"`
	UptimeSeconds uint64  `json:"uptime_seconds"`
}

// ClusterResources consolida o consumo de recursos agregados.
type ClusterResources struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsedBytes    uint64  `json:"memory_used_bytes"`
	MemoryTotalBytes   uint64  `json:"memory_total_bytes"`
	MemoryUsagePercent float32 `json:"memory_usage_percent"`
}

// ClusterOverview é o snapshot consolidado de toda a infraestrutura
type ClusterOverview struct {
	ClusterName     string           `json:"cluster_name"`
	Environment     string           `json:"environment"`
	UptimeSeconds   uint64           `json:"uptime_seconds"`
	LastUpdated     time.Time        `json:"last_updated"`
	Resources       ClusterResources `json:"resources"`
	TotalNodes 		int				 `json:"total_nodes"`
	HealthyNodes    int              `json:"healthy_nodes"`
	TotalServices   int              `json:"total_services"`
	HealthyServices int              `json:"healthy_services"`
}

// TerminalCommandRequest representa a entrada vinda do input do terminal web
type TerminalCommandRequest struct {
	Command string `json:"command"`
	Args    []string `json:"args"`
}

// TerminalCommandResponse encapsula a resposta processada para o frontend
type TerminalCommandResponse struct {
	Output     string      `json:"output"`
	Timestamp  time.Time   `json:"timestamp"`
	OutputText string      `json:"output_text"`
	Data       interface{} `json:"data,omitempty"`
}

