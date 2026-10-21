package collector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"homelab-telemetry-api/internal/domain"
	"homelab-telemetry-api/internal/infra/collector"
)

func TestHTTPServiceCollector_CollectorServices(t *testing.T) {
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyServer.Close()

	downServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer downServer.Close()

	targets := []collector.HTTPTarget{
		{Name: "auth-service", URL: healthyServer.URL},
		{Name: "broken-service", URL: downServer.URL},
		{Name: "offline-service", URL: "http://localhost:19999"},
	}

	probe := collector.NewHTTPServiceCollector("homelab-probes", targets, 1*time.Second)

	ctx := context.Background()
	// Chamando CollectServices exatamente como definido na sua interface
	services, err := probe.CollectServices(ctx)
	if err != nil {
		t.Fatalf("unexpected error during collection: %v", err)
	}

	if len(services) != 3 {
		t.Fatalf("expected 3 services checked, got %d", len(services))
	}

	results := make(map[string]domain.ServiceTelemetry)
	for _, s := range services {
		results[s.Name] = s
	}

	if auth, ok := results["auth-service"]; !ok {
		t.Errorf("auth-service missing from results")
	} else {
		if auth.Status != domain.StatusHealthy {
			t.Errorf("expected auth-service to be UP, got %s", auth.Status)
		}
		if auth.LatencyMs < 0 {
			t.Errorf("invalid latency measured: %d", auth.LatencyMs)
		}
	}

	if broken, ok := results["broken-service"]; !ok {
		t.Errorf("broken-service missing from results")
	} else {
		if broken.Status != domain.StatusDown {
			t.Errorf("expected broken-service to be DOWN, got %s", broken.Status)
		}
	}

	if offline, ok := results["offline-service"]; !ok {
		t.Errorf("offline-service missing from results")
	} else {
		if offline.Status != domain.StatusDown {
			t.Errorf("expected offline-service to be DOWN, got %s", offline.Status)
		}
	}
}