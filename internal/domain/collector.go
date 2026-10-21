package domain

import "context"

// ServiceCollector faz a sondagem de um ou mais serviços de aplicação
type ServiceCollector interface {
	Name() string
	CollectServices(ctx context.Context) ([]ServiceTelemetry, error)
}

// NodeCollector faz a sondagem de um ou mais nós do homelab
type NodeCollector interface {
	Name() string
	CollectorNodes(ctx context.Context) ([]NodeTelemetry, error)
}