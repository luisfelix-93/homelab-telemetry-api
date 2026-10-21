package collector

import (
	"context"
	"homelab-telemetry-api/internal/domain"
	"net/http"
	"sync"
	"time"
)

// HTTPTarget define o alvo do probe HTTP
type HTTPTarget struct {
	Name string
	URL  string
}

type HTTPServiceCollector struct {
	name    string
	targets []HTTPTarget
	client  *http.Client
}

var _ domain.ServiceCollector = (*HTTPServiceCollector)(nil)

func NewHTTPServiceCollector(name string, targets []HTTPTarget, timeout time.Duration) *HTTPServiceCollector {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &HTTPServiceCollector{
		name: name,
		targets: targets,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}
func (c *HTTPServiceCollector) Name() string {
	return c.name
}

func (c *HTTPServiceCollector) CollectServices(ctx context.Context) ([]domain.ServiceTelemetry, error){
	if len(c.targets) == 0 {
		return []domain.ServiceTelemetry{}, nil
	}

	results := make([]domain.ServiceTelemetry, len(c.targets))
	var wg sync.WaitGroup
	wg.Add(len(c.targets))

	for i, target := range c.targets {
		go func(idx int, t HTTPTarget) {
			defer wg.Done()
			start := time.Now()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
			
			telemetry := domain.ServiceTelemetry{
				Name: t.Name,
				LastCheck: time.Now().UTC(),
			}

			if err != nil {
				telemetry.Status = domain.StatusDown
				telemetry.Message = err.Error()
				results[idx] = telemetry
				return
			}

			resp, err := c.client.Do(req)
			if err != nil {
				telemetry.Status = domain.StatusDown
				telemetry.Message = err.Error()
				results[idx] = telemetry
				return
			}
			defer resp.Body.Close()

			duration := time.Since(start)
			telemetry.LatencyMs = duration.Milliseconds()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				telemetry.Status = domain.StatusHealthy
			} else if resp.StatusCode >= 500 {
				telemetry.Status = domain.StatusDown
				telemetry.Message = "unhealthy status code"
			} else {
				telemetry.Status = domain.StatusDegraded
				telemetry.Message = "unhealthy status code"
			}
			results[idx] = telemetry
		}(i, target)
	}

	wg.Wait()
	return results, nil
}