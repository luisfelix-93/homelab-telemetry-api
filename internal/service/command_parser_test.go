package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"homelab-telemetry-api/internal/domain"
	"homelab-telemetry-api/internal/service"
)

type mockSnapshotRepository struct {
	overview    *domain.ClusterOverview
	overviewErr error
	services    []domain.ServiceTelemetry
	servicesErr error
	nodes       []domain.NodeTelemetry
	nodesErr    error
}

func (m *mockSnapshotRepository) SaveOverView(ctx context.Context, overview *domain.ClusterOverview) error {
	return nil
}

func (m *mockSnapshotRepository) GetOverView(ctx context.Context) (*domain.ClusterOverview, error) {
	return m.overview, m.overviewErr
}

func (m *mockSnapshotRepository) SaveServices(ctx context.Context, services []domain.ServiceTelemetry) error {
	return nil
}

func (m *mockSnapshotRepository) GetServices(ctx context.Context) ([]domain.ServiceTelemetry, error) {
	return m.services, m.servicesErr
}

func (m *mockSnapshotRepository) SaveNodes(ctx context.Context, nodes []domain.NodeTelemetry) error {
	return nil
}

func (m *mockSnapshotRepository) GetNodes(ctx context.Context) ([]domain.NodeTelemetry, error) {
	return m.nodes, m.nodesErr
}

func TestNewCommandParser(t *testing.T) {
	repo := &mockSnapshotRepository{}
	parser := service.NewCommandParser(repo)

	if parser == nil {
		t.Fatal("expected non-nil CommandParser")
	}
}

func TestCommandParser_Execute_Help(t *testing.T) {
	ctx := context.Background()
	repo := &mockSnapshotRepository{}
	parser := service.NewCommandParser(repo)

	testCases := []struct {
		name    string
		command string
	}{
		{name: "lowercase", command: "help"},
		{name: "uppercase and spaced", command: "  HELP  "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := domain.TerminalCommandRequest{Command: tc.command}
			resp, err := parser.Execute(ctx, req)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatal("expected non-nil response")
			}
			if resp.Command != "help" {
				t.Errorf("expected command 'help', got '%s'", resp.Command)
			}
			if resp.Timestamp.IsZero() {
				t.Error("expected non-zero timestamp")
			}
			if !strings.Contains(resp.OutputText, "Available commands:") {
				t.Errorf("expected help output text, got: %s", resp.OutputText)
			}

			dataMap, ok := resp.Data.(map[string]interface{})
			if !ok {
				t.Fatalf("expected map[string]interface{} in resp.Data, got %T", resp.Data)
			}
			commands, ok := dataMap["commands"].([]string)
			if !ok {
				t.Fatalf("expected []string for commands in resp.Data, got %T", dataMap["commands"])
			}
			if len(commands) != 5 {
				t.Errorf("expected 5 available commands, got %d", len(commands))
			}
		})
	}
}

func TestCommandParser_Execute_Status(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		overview := &domain.ClusterOverview{
			ClusterName:     "k3s-cluster",
			Environment:     "staging",
			UptimeSeconds:   7200,
			TotalNodes:      3,
			HealthyNodes:    3,
			TotalServices:   10,
			HealthyServices: 9,
			Resources: domain.ClusterResources{
				CPUUsagePercent:    45.5,
				MemoryUsagePercent: 62.3,
			},
		}

		repo := &mockSnapshotRepository{overview: overview}
		parser := service.NewCommandParser(repo)

		resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "status"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.Command != "status" {
			t.Errorf("expected command 'status', got '%s'", resp.Command)
		}
		if !strings.Contains(resp.OutputText, "Cluster: k3s-cluster (staging)") {
			t.Errorf("expected cluster info in OutputText, got: %s", resp.OutputText)
		}
		if !strings.Contains(resp.OutputText, "CPU Usage: 45.5% | Memory Usage: 62.3%") {
			t.Errorf("expected resource usage in OutputText, got: %s", resp.OutputText)
		}
		if resp.Data != overview {
			t.Errorf("expected resp.Data to match overview")
		}
	})

	t.Run("repository error or snapshot not ready", func(t *testing.T) {
		repo := &mockSnapshotRepository{overviewErr: domain.ErrSnapshotNotReady}
		parser := service.NewCommandParser(repo)

		resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "status"})
		if err != nil {
			t.Fatalf("expected nil error (handled gracefully), got: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		expectedText := "Error: Cluster telemetry snapshot not ready yet."
		if resp.OutputText != expectedText {
			t.Errorf("expected OutputText '%s', got '%s'", expectedText, resp.OutputText)
		}
		if resp.Data != nil {
			t.Errorf("expected nil Data, got %v", resp.Data)
		}
	})
}

func TestCommandParser_Execute_Services(t *testing.T) {
	ctx := context.Background()

	t.Run("success with monitored services", func(t *testing.T) {
		checkTime := time.Date(2026, 10, 25, 12, 30, 0, 0, time.UTC)
		servicesList := []domain.ServiceTelemetry{
			{
				Name:      "api-gateway",
				Status:    domain.StatusHealthy,
				LatencyMs: 15,
				LastCheck: checkTime,
			},
			{
				Name:      "postgres",
				Status:    domain.StatusDegraded,
				LatencyMs: 120,
				LastCheck: checkTime,
			},
		}

		repo := &mockSnapshotRepository{services: servicesList}
		parser := service.NewCommandParser(repo)

		resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "  SERVICES "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.Command != "services" {
			t.Errorf("expected command 'services', got '%s'", resp.Command)
		}
		if !strings.Contains(resp.OutputText, "NAME") || !strings.Contains(resp.OutputText, "LATENCY") {
			t.Errorf("expected table headers in OutputText, got: %s", resp.OutputText)
		}
		if !strings.Contains(resp.OutputText, "api-gateway") || !strings.Contains(resp.OutputText, "15") {
			t.Errorf("expected api-gateway row in OutputText, got: %s", resp.OutputText)
		}
		if !strings.Contains(resp.OutputText, "postgres") || !strings.Contains(resp.OutputText, "120") {
			t.Errorf("expected postgres row in OutputText, got: %s", resp.OutputText)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockSnapshotRepository{servicesErr: errors.New("storage error")}
		parser := service.NewCommandParser(repo)

		resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "services"})
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		expectedText := "No services monitored or snapshot not ready."
		if resp.OutputText != expectedText {
			t.Errorf("expected OutputText '%s', got '%s'", expectedText, resp.OutputText)
		}
	})

	t.Run("empty services list", func(t *testing.T) {
		repo := &mockSnapshotRepository{services: []domain.ServiceTelemetry{}}
		parser := service.NewCommandParser(repo)

		resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "services"})
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		expectedText := "No services monitored or snapshot not ready."
		if resp.OutputText != expectedText {
			t.Errorf("expected OutputText '%s', got '%s'", expectedText, resp.OutputText)
		}
	})
}

func TestCommandParser_Execute_Nodes(t *testing.T) {
	ctx := context.Background()

	t.Run("success with registered nodes", func(t *testing.T) {
		nodesList := []domain.NodeTelemetry{
			{
				ID:            "master-01",
				Role:          "control-plane",
				Status:        "Ready",
				CPUPercent:    18.5,
				MemoryUsedMB:  4096,
				MemoryTotalMB: 16384,
			},
			{
				ID:            "worker-01",
				Role:          "worker",
				Status:        "Ready",
				CPUPercent:    62.0,
				MemoryUsedMB:  12288,
				MemoryTotalMB: 16384,
			},
		}

		repo := &mockSnapshotRepository{nodes: nodesList}
		parser := service.NewCommandParser(repo)

		resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "nodes"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.Command != "nodes" {
			t.Errorf("expected command 'nodes', got '%s'", resp.Command)
		}
		if !strings.Contains(resp.OutputText, "ID") || !strings.Contains(resp.OutputText, "RAM (USED/TOT)") {
			t.Errorf("expected table headers in OutputText, got: %s", resp.OutputText)
		}
		if !strings.Contains(resp.OutputText, "master-01") || !strings.Contains(resp.OutputText, "4096MB/16384MB") {
			t.Errorf("expected master-01 row in OutputText, got: %s", resp.OutputText)
		}
		if !strings.Contains(resp.OutputText, "worker-01") || !strings.Contains(resp.OutputText, "12288MB/16384MB") {
			t.Errorf("expected worker-01 row in OutputText, got: %s", resp.OutputText)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockSnapshotRepository{nodesErr: errors.New("storage error")}
		parser := service.NewCommandParser(repo)

		resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "nodes"})
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		expectedText := "No nodes registered or snapshot not ready."
		if resp.OutputText != expectedText {
			t.Errorf("expected OutputText '%s', got '%s'", expectedText, resp.OutputText)
		}
	})

	t.Run("empty nodes list", func(t *testing.T) {
		repo := &mockSnapshotRepository{nodes: []domain.NodeTelemetry{}}
		parser := service.NewCommandParser(repo)

		resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "nodes"})
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		expectedText := "No nodes registered or snapshot not ready."
		if resp.OutputText != expectedText {
			t.Errorf("expected OutputText '%s', got '%s'", expectedText, resp.OutputText)
		}
	})
}

func TestCommandParser_Execute_Clear(t *testing.T) {
	ctx := context.Background()
	repo := &mockSnapshotRepository{}
	parser := service.NewCommandParser(repo)

	resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: "clear"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Command != "clear" {
		t.Errorf("expected command 'clear', got '%s'", resp.Command)
	}
	if resp.OutputText != "" {
		t.Errorf("expected empty OutputText, got '%s'", resp.OutputText)
	}
}

func TestCommandParser_Execute_UnknownCommand(t *testing.T) {
	ctx := context.Background()
	repo := &mockSnapshotRepository{}
	parser := service.NewCommandParser(repo)

	testCases := []struct {
		name    string
		command string
	}{
		{name: "invalid command", command: "reboot"},
		{name: "empty string", command: ""},
		{name: "spaces only", command: "   "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := parser.Execute(ctx, domain.TerminalCommandRequest{Command: tc.command})
			if !errors.Is(err, domain.ErrCommandNotFound) {
				t.Fatalf("expected ErrCommandNotFound, got: %v", err)
			}
			if resp == nil {
				t.Fatal("expected non-nil response")
			}
			trimmed := strings.ToLower(strings.TrimSpace(tc.command))
			expectedText := "Command not found: '" + trimmed + "'. Type 'help' for available commands."
			if resp.OutputText != expectedText {
				t.Errorf("expected OutputText '%s', got '%s'", expectedText, resp.OutputText)
			}
		})
	}
}
