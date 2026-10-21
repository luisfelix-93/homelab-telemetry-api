package cache

import (
	"context"
	"errors"
	"homelab-telemetry-api/internal/domain"
	"sync"
)

var _ domain.SnapshotRepository = (*MemorySnapshotRepository)(nil)

type MemorySnapshotRepository struct {
	mu       sync.RWMutex
	overview *domain.ClusterOverview
	services []domain.ServiceTelemetry
	nodes    []domain.NodeTelemetry
}

func NewMemorySnapshotRepository() *MemorySnapshotRepository {
	return &MemorySnapshotRepository{}
}

func (r *MemorySnapshotRepository) SaveOverView(ctx context.Context, overview *domain.ClusterOverview) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if overview == nil {
		return errors.New("overview cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Cópia defensiva por valor
	copied := *overview
	r.overview = &copied
	return nil
}

func (r *MemorySnapshotRepository) GetOverView(ctx context.Context) (*domain.ClusterOverview, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.overview == nil {
		return nil, domain.ErrSnapshotNotReady
	}

	copied := *r.overview
	return &copied, nil
}

func (r *MemorySnapshotRepository) SaveServices(ctx context.Context, services []domain.ServiceTelemetry) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if services == nil {
		return errors.New("services cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	copied := make([]domain.ServiceTelemetry, len(services))
	copy(copied, services)
	r.services = copied
	return nil
}

func (r *MemorySnapshotRepository) GetServices(ctx context.Context) ([]domain.ServiceTelemetry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.services == nil {
		return nil, domain.ErrSnapshotNotReady
	}

	copied := make([]domain.ServiceTelemetry, len(r.services))
	copy(copied, r.services)

	return copied, nil
}

func (r *MemorySnapshotRepository) SaveNodes(ctx context.Context, nodes []domain.NodeTelemetry) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if nodes == nil {
		return errors.New("nodes cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	copied := make([]domain.NodeTelemetry, len(nodes))
	copy(copied, nodes)
	r.nodes = copied
	return nil
}

func (r *MemorySnapshotRepository) GetNodes(ctx context.Context) ([]domain.NodeTelemetry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.nodes == nil {
		return nil, domain.ErrSnapshotNotReady
	}

	copied := make([]domain.NodeTelemetry, len(r.nodes))
	copy(copied, r.nodes)
	return copied, nil
}