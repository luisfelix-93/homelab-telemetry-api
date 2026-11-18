package collector_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"homelab-telemetry-api/internal/domain"
	"homelab-telemetry-api/internal/infra/collector"
)

func TestK3sServiceCollector_CollectorServices(t *testing.T) {
	t.Run("should successfully collect services and map health status based on active endpoints", func(t *testing.T) {
		// 1. Simula serviços e endpoints no Fake Clientset
		fakeClient := fake.NewSimpleClientset(
			// Serviço saudável com endpoints ativos
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "auth-service",
					Namespace: "mcp-system",
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "auth-service",
					Namespace: "mcp-system",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.42.0.15"},
						},
					},
				},
			},

			// Serviço sem endpoints ativos (Down / Sem pods rodando)
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "postgres-service",
					Namespace: "mcp-system",
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "postgres-service",
					Namespace: "mcp-system",
				},
				Subsets: []corev1.EndpointSubset{}, // Sem endereços ativos
			},
		)

		// Instancia o coletor passando namespace vazio (para buscar em todos) e o cliente fake
		serviceCollector := collector.NewK3sServiceCollector("k3s-service-probe", fakeClient, "")

		ctx := context.Background()
		services, err := serviceCollector.CollectServices(ctx)
		if err != nil {
			t.Fatalf("unexpected error collecting k3s services: %v", err)
		}

		if len(services) != 2 {
			t.Fatalf("expected 2 services, got %d", len(services))
		}

		// Mapeia resultados para validação
		serviceMap := make(map[string]domain.ServiceTelemetry)
		for _, s := range services {
			serviceMap[s.Name] = s
		}

		// Valida auth-service (deve estar UP)
		authSvc, ok := serviceMap["mcp-system/auth-service"]
		if !ok {
			t.Fatalf("auth-service not found in telemetry results")
		}
		if authSvc.Status != domain.StatusHealthy {
			t.Errorf("expected auth-service status to be UP, got %s", authSvc.Status)
		}

		// Valida postgres-service (deve estar DOWN por falta de endpoints)
		pgSvc, ok := serviceMap["mcp-system/postgres-service"]
		if !ok {
			t.Fatalf("postgres-service not found in telemetry results")
		}
		if pgSvc.Status != domain.StatusDown {
			t.Errorf("expected postgres-service status to be DOWN, got %s", pgSvc.Status)
		}
	})
}