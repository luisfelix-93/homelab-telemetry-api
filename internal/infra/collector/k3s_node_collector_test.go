package collector

import (
	"context"
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	metricsv1 "k8s.io/metrics/pkg/apis/metrics/v1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

func TestK3sNodeCollector_Name(t *testing.T) {
	nodeCollector := NewK3sNodeCollectorWithClient("k3s-cluster-probe", fake.NewSimpleClientset())

	if got := nodeCollector.Name(); got != "k3s-cluster-probe" {
		t.Fatalf("Name() = %q, want %q", got, "k3s-cluster-probe")
	}
}

func TestK3sNodeCollector_CollectNodes(t *testing.T) {
	t.Run("collects node roles, status, capacity, and usage metrics", func(t *testing.T) {
		client := fake.NewSimpleClientset(
			testNode("homelab-master-01", map[string]string{
				"node-role.kubernetes.io/control-plane": "true",
			}, corev1.ConditionTrue, "4", "16Gi"),
			testNode("homelab-worker-01", map[string]string{
				"node-role.kubernetes.io/master": "true",
			}, corev1.ConditionFalse, "8", "32Gi"),
		)
		metricsClient := testMetricsClient(
			testNodeMetrics("homelab-master-01", "500m", "4Gi"),
			testNodeMetrics("homelab-worker-01", "2", "8Gi"),
		)
		nodeCollector := &K3sNodeCollector{
			name:          "k3s-cluster-probe",
			clientset:     client,
			metricsClient: metricsClient,
		}

		nodes, err := nodeCollector.CollectNodes(context.Background())
		if err != nil {
			t.Fatalf("CollectNodes() error = %v", err)
		}
		if len(nodes) != 2 {
			t.Fatalf("CollectNodes() returned %d nodes, want 2", len(nodes))
		}

		gotByID := make(map[string]struct {
			role          string
			status        string
			cpuCores      int
			cpuPercent    float64
			memoryTotalMB int64
			memoryUsedMB  int64
			uptimeSeconds uint64
		}, len(nodes))
		for _, node := range nodes {
			gotByID[node.ID] = struct {
				role          string
				status        string
				cpuCores      int
				cpuPercent    float64
				memoryTotalMB int64
				memoryUsedMB  int64
				uptimeSeconds uint64
			}{
				role:          node.Role,
				status:        node.Status,
				cpuCores:      node.CPUCores,
				cpuPercent:    node.CPUPercent,
				memoryTotalMB: node.MemoryTotalMB,
				memoryUsedMB:  node.MemoryUsedMB,
				uptimeSeconds: node.UptimeSeconds,
			}
		}

		tests := []struct {
			name          string
			role          string
			status        string
			cpuCores      int
			cpuPercent    float64
			memoryTotalMB int64
			memoryUsedMB  int64
		}{
			{
				name:          "homelab-master-01",
				role:          "control-plane",
				status:        "Ready",
				cpuCores:      4,
				cpuPercent:    0.125,
				memoryTotalMB: 16384,
				memoryUsedMB:  4096,
			},
			{
				name:          "homelab-worker-01",
				role:          "control-plane",
				status:        "NotReady",
				cpuCores:      8,
				cpuPercent:    0.25,
				memoryTotalMB: 32768,
				memoryUsedMB:  8192,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				node, ok := gotByID[test.name]
				if !ok {
					t.Fatalf("node %q missing from result", test.name)
				}
				if node.role != test.role {
					t.Errorf("role = %q, want %q", node.role, test.role)
				}
				if node.status != test.status {
					t.Errorf("status = %q, want %q", node.status, test.status)
				}
				if node.cpuCores != test.cpuCores {
					t.Errorf("cpuCores = %d, want %d", node.cpuCores, test.cpuCores)
				}
				if node.cpuPercent != test.cpuPercent {
					t.Errorf("cpuPercent = %v, want %v", node.cpuPercent, test.cpuPercent)
				}
				if node.memoryTotalMB != test.memoryTotalMB {
					t.Errorf("memoryTotalMB = %d, want %d", node.memoryTotalMB, test.memoryTotalMB)
				}
				if node.memoryUsedMB != test.memoryUsedMB {
					t.Errorf("memoryUsedMB = %d, want %d", node.memoryUsedMB, test.memoryUsedMB)
				}
				if node.uptimeSeconds != 0 {
					t.Errorf("uptimeSeconds = %d, want 0", node.uptimeSeconds)
				}
			})
		}
	})

	t.Run("defaults to worker and zero metrics when metrics are unavailable", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-without-metrics"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionUnknown},
				},
			},
		}
		nodeCollector := NewK3sNodeCollectorWithClient("k3s-cluster-probe", fake.NewSimpleClientset(node))

		nodes, err := nodeCollector.CollectNodes(context.Background())
		if err != nil {
			t.Fatalf("CollectNodes() error = %v", err)
		}
		if len(nodes) != 1 {
			t.Fatalf("CollectNodes() returned %d nodes, want 1", len(nodes))
		}

		got := nodes[0]
		if got.Role != "worker" || got.Status != "NotReady" {
			t.Errorf("role/status = %q/%q, want %q/%q", got.Role, got.Status, "worker", "NotReady")
		}
		if got.CPUCores != 0 || got.CPUPercent != 0 {
			t.Errorf("CPU metrics = %d cores, %v percent, want 0/0", got.CPUCores, got.CPUPercent)
		}
		if got.MemoryTotalMB != 0 || got.MemoryUsedMB != 0 {
			t.Errorf("memory metrics = %d/%d MB, want 0/0", got.MemoryTotalMB, got.MemoryUsedMB)
		}
	})

	t.Run("continues collecting nodes when the metrics API fails", func(t *testing.T) {
		client := fake.NewSimpleClientset(testNode("node-1", nil, corev1.ConditionTrue, "2", "2Gi"))
		metricsClient := metricsfake.NewSimpleClientset()
		metricsClient.PrependReactor("list", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("metrics API unavailable")
		})
		nodeCollector := &K3sNodeCollector{
			clientset:     client,
			metricsClient: metricsClient,
		}

		nodes, err := nodeCollector.CollectNodes(context.Background())
		if err != nil {
			t.Fatalf("CollectNodes() error = %v, want nil when metrics API fails", err)
		}
		if len(nodes) != 1 {
			t.Fatalf("CollectNodes() returned %d nodes, want 1", len(nodes))
		}
		if nodes[0].CPUPercent != 0 || nodes[0].MemoryUsedMB != 0 {
			t.Errorf("usage metrics = %v CPU, %d MB memory, want 0/0", nodes[0].CPUPercent, nodes[0].MemoryUsedMB)
		}
	})

	t.Run("returns an error when the node list fails", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.PrependReactor("list", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("kubernetes API unavailable")
		})
		nodeCollector := NewK3sNodeCollectorWithClient("k3s-cluster-probe", client)

		nodes, err := nodeCollector.CollectNodes(context.Background())
		if err == nil {
			t.Fatal("CollectNodes() error = nil, want an error")
		}
		if !strings.Contains(err.Error(), "failed to list k3s nodes") {
			t.Errorf("CollectNodes() error = %q, want contextual error", err)
		}
		if nodes != nil {
			t.Errorf("CollectNodes() nodes = %v, want nil on list failure", nodes)
		}
	})
}

func testNode(name string, labels map[string]string, ready corev1.ConditionStatus, cpu, memory string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: ready},
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
			},
		},
	}
}

func testNodeMetrics(name, cpu, memory string) *metricsv1.NodeMetrics {
	return &metricsv1.NodeMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(memory),
		},
	}
}

func testMetricsClient(nodeMetrics ...*metricsv1.NodeMetrics) *metricsfake.Clientset {
	client := metricsfake.NewSimpleClientset()
	items := make([]metricsv1.NodeMetrics, 0, len(nodeMetrics))
	for _, metric := range nodeMetrics {
		items = append(items, *metric)
	}
	client.PrependReactor("list", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &metricsv1.NodeMetricsList{Items: items}, nil
	})
	return client
}
