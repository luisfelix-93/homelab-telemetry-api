package collector

import (
	"context"
	"fmt"
	"homelab-telemetry-api/internal/domain"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K3sServiceCollector struct {
	name      string
	clientset kubernetes.Interface
	namespace string // se vazio, busca em todos os namespaces
}

var _ domain.ServiceCollector = (*K3sServiceCollector)(nil)

func NewK3sServiceCollector(name string, clientset kubernetes.Interface, namespace string) *K3sServiceCollector {
	return &K3sServiceCollector{
		name:      name,
		clientset: clientset,
		namespace: namespace,
	}
}

func (k *K3sServiceCollector) Name() string {
	return k.name
}

func (k *K3sServiceCollector) CollectServices(ctx context.Context) ([]domain.ServiceTelemetry, error) {
	// Se o namespace estiver vazio, usa "" para listar todos
	ns := k.namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	services, err := k.clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list k3s services: %w", err)
	}
	var telemetryServices []domain.ServiceTelemetry
	
	for _, svc := range services.Items {
		// Ignora serviços internos do sistema kubernetes se desejar
		if svc.Namespace == "kube-system" && svc.Name == "kubernetes" {
			continue
		}

		// verifica se há endpoints ativos para este serviço
		endpoints, err := k.clientset.CoreV1().Endpoints(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{})

		status := domain.StatusHealthy
		message := "service online"
		var activeSubsets int

		if err == nil {
			for _, subset := range endpoints.Subsets {
				if len(subset.Addresses) > 0 {
					activeSubsets += len(subset.Addresses)
				}
			}
		}

		if err != nil {
			status = domain.StatusDegraded
			message = fmt.Sprintf("failed to get endpoints for %s/%s: %v", svc.Namespace, svc.Name, err)
		} else if activeSubsets == 0 {
			status = domain.StatusDown
			message = "no active endpoints (pods) available"
		}

		// Formata o nome para a exibição limpa no terminal (ex: "mcp-system/postgres")
		displayName := svc.Name
		if k.namespace == "" && svc.Namespace != "default" {
			displayName = fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
		}

		telemetryServices = append(telemetryServices, domain.ServiceTelemetry{
			Name:       displayName,
			Status:     status,
			LatencyMs:  2, // Latencia interna do plano de controle do k3s
			Message:    message,
		})

	}

	return telemetryServices, nil
}

