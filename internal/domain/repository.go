package domain

import "context"

// SnapeshotRepository define as operações de leitura e escrita do estado em cache
type SnapshotRepository interface {
	// Overview
	SaveOverView(ctx context.Context, overview *ClusterOverview) error
	GetOverView(ctx context.Context) (*ClusterOverview, error)

	// Services
	SaveServices(ctx context.Context, services []ServiceTelemetry) error
	GetServices(ctx context.Context)([]ServiceTelemetry, error)

	// Nodes 
	SaveNodes(ctx context.Context, nodes []NodeTelemetry) error
	GetNodes(ctx context.Context)([]NodeTelemetry, error)

}