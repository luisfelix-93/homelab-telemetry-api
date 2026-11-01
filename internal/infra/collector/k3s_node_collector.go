package collector

import (
	"context"
	"fmt"
	"homelab-telemetry-api/internal/domain"

	"path/filepath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type K3sNodeCollector struct {
	name      string
	clientset kubernetes.Interface
	metricsClient metricsv.Interface
}	

var _ domain.NodeCollector = (*K3sNodeCollector)(nil)

func NewK3sNodeCollector(name string) (*K3sNodeCollector, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		var kubeconfigPath string
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		} else {
			kubeconfigPath = "/etc/rancher/k3s/k3s.yaml"
		}
		
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load k3s kubeconfig: %w", err)
		}

		
		
	}

	clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
		}
	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		metricsClient = nil
	}
	return &K3sNodeCollector{
			name:      name,
			clientset: clientset,
			metricsClient: metricsClient,
	}, nil		
}

func NewK3sNodeCollectorWithClient(name string, clientset kubernetes.Interface) *K3sNodeCollector {
	return &K3sNodeCollector{
		name:      name,
		clientset: clientset,
	}
}

func (k *K3sNodeCollector) Name() string {
	return k.name
}

func (k *K3sNodeCollector) CollectNodes(ctx context.Context) ([]domain.NodeTelemetry, error) {
	nodeList, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list k3s nodes: %w", err)
	}

	metricsMap := make(map[string]nodeMetricData)
	if k.metricsClient != nil {
		if nodeMetricsList, err := k.metricsClient.MetricsV1().NodeMetricses().List(ctx, metav1.ListOptions{}); err == nil {
			for _, m := range nodeMetricsList.Items {
				cpuUsage := m.Usage.Cpu().MilliValue()
				memUsageBytes := m.Usage.Memory().Value()
				metricsMap[m.Name] = nodeMetricData{
					cpuMilliUsage: cpuUsage,
					memUsedBytes: memUsageBytes,
				}
			}
		}
	}
	var telemetryNodes []domain.NodeTelemetry

	for _, node := range nodeList.Items {
		// Determina o papel do nó (Control-Plane vs Worker) pelas labels do K3S/Kubernetes
		role := "worker"
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			role = "control-plane"
		} else if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			role = "control-plane"
		}

		// Status do nó
		status := "NotReady"
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				status = "Ready"
				break
			}
		}

		cpuAllocatable := node.Status.Allocatable.Cpu()
		memAllocatable := node.Status.Allocatable.Memory()

		var cpuCores int
		if cpuAllocatable != nil {
			cpuCores = int(cpuAllocatable.Value())
		}

		var memTotalMB int64
		if memAllocatable != nil {
			memTotalMB = memAllocatable.Value() / (1024 * 1024)
		}

		var cpuPercent float64
		var memUsedMB int64

		if metric, ok := metricsMap[node.Name]; ok {
			memUsedMB = metric.memUsedBytes / (1024 * 1024)

			totalMilicores := int64(cpuCores * 1000)
			if totalMilicores > 0 {
				cpuPercent = (float64(metric.cpuMilliUsage) / float64(totalMilicores))
			}
		}

		telemetryNodes = append(telemetryNodes, domain.NodeTelemetry{
			ID:            node.Name,
			Role:          role,
			Status:        status,
			CPUCores:      cpuCores,
			CPUPercent:    cpuPercent,
			MemoryTotalMB: memTotalMB,
			MemoryUsedMB:  memUsedMB,
			UptimeSeconds: 0,
		})
	}

	return telemetryNodes, nil
}


type nodeMetricData struct {
	cpuMilliUsage int64
	memUsedBytes  int64
}