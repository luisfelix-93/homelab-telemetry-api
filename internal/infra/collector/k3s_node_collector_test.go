package collector_test

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"homelab-telemetry-api/internal/infra/collector"
)

func TestK3sNodeCollector_CollectNodes(t *testing.T) {
	t.Run("should successfully collect control-plane and worker nodes from k3s fake client", func(t *testing.T) {
		// Inicializa o fake clientset com nós simulados
		fakeClient := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "homelab-master-01",
					Labels: map[string]string{
						"node-role.kubernetes.io/control-plane": "true",
					},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewQuantity(4, resource.DecimalSI),
						corev1.ResourceMemory: *resource.NewQuantity(16*1024*1024*1024, resource.BinarySI), // 16 GiB
					},
				},
			},
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "homelab-worker-01",
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "true",
					},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse}, // Simulando nó NotReady
					},
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewQuantity(8, resource.DecimalSI),
						corev1.ResourceMemory: *resource.NewQuantity(32*1024*1024*1024, resource.BinarySI), // 32 GiB
					},
				},
			},
		)

		collectorInstance := collector.NewK3sNodeCollectorWithClient("k3s-cluster-probe", fakeClient)

		nodes, err := collectorInstance.CollectNodes(context.Background())
		if err != nil {
			t.Fatalf("unexpected error collecting k3s nodes: %v", err)
		}

		if len(nodes) != 2 {
			t.Fatalf("expected 2 nodes, got %d", len(nodes))
		}

		nodeMap := make(map[string]domainNodeCheck)
		for _, n := range nodes {
			nodeMap[n.ID] = domainNodeCheck{
				role:          n.Role,
				status:        n.Status,
				cpuCores:      n.CPUCores,
				memoryTotalMB: n.MemoryTotalMB,
			}
		}

		// Valida Master
		master, ok := nodeMap["homelab-master-01"]
		if !ok {
			t.Fatalf("master node missing")
		}
		if master.role != "control-plane" {
			t.Errorf("expected role 'control-plane', got '%s'", master.role)
		}
		if master.status != "Ready" {
			t.Errorf("expected status 'Ready', got '%s'", master.status)
		}
		if master.cpuCores != 4 || master.memoryTotalMB != 16384 {
			t.Errorf("unexpected master specs: CPU=%d, RAM=%dMB", master.cpuCores, master.memoryTotalMB)
		}

		// Valida Worker
		worker, ok := nodeMap["homelab-worker-01"]
		if !ok {
			t.Fatalf("worker node missing")
		}
		if worker.role != "worker" {
			t.Errorf("expected role 'worker', got '%s'", worker.role)
		}
		if worker.status != "NotReady" {
			t.Errorf("expected status 'NotReady', got '%s'", worker.status)
		}
		if worker.cpuCores != 8 || worker.memoryTotalMB != 32768 {
			t.Errorf("unexpected worker specs: CPU=%d, RAM=%dMB", worker.cpuCores, worker.memoryTotalMB)
		}
	})
}

type domainNodeCheck struct {
	role          string
	status        string
	cpuCores      int
	memoryTotalMB int64
}