package transporthttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"homelab-telemetry-api/internal/domain"
	"homelab-telemetry-api/internal/service"
	transporthttp "homelab-telemetry-api/internal/transport/http"
)

type mockSnapshotRepo struct {
	overview    *domain.ClusterOverview
	overviewErr error
	services    []domain.ServiceTelemetry
	servicesErr error
	nodes       []domain.NodeTelemetry
	nodesErr    error
}

func (m *mockSnapshotRepo) SaveOverView(ctx context.Context, overview *domain.ClusterOverview) error {
	return nil
}

func (m *mockSnapshotRepo) GetOverView(ctx context.Context) (*domain.ClusterOverview, error) {
	return m.overview, m.overviewErr
}

func (m *mockSnapshotRepo) SaveServices(ctx context.Context, services []domain.ServiceTelemetry) error {
	return nil
}

func (m *mockSnapshotRepo) GetServices(ctx context.Context) ([]domain.ServiceTelemetry, error) {
	return m.services, m.servicesErr
}

func (m *mockSnapshotRepo) SaveNodes(ctx context.Context, nodes []domain.NodeTelemetry) error {
	return nil
}

func (m *mockSnapshotRepo) GetNodes(ctx context.Context) ([]domain.NodeTelemetry, error) {
	return m.nodes, m.nodesErr
}

func setupTestHandler(repo *mockSnapshotRepo) *transporthttp.Handler {
	cmdParser := service.NewCommandParser(repo)
	return transporthttp.NewHandler(repo, cmdParser)
}

func TestHandler_HealthCheck(t *testing.T) {
	repo := &mockSnapshotRepo{}
	handler := setupTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", body["status"])
	}
}

func TestHandler_GetOverView(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		overview := &domain.ClusterOverview{
			ClusterName:   "test-cluster",
			Environment:   "production",
			UptimeSeconds: 3600,
		}
		repo := &mockSnapshotRepo{overview: overview}
		handler := setupTestHandler(repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/overview", nil)
		rec := httptest.NewRecorder()

		handler.GetOverView(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var result domain.ClusterOverview
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if result.ClusterName != "test-cluster" {
			t.Errorf("expected ClusterName 'test-cluster', got '%s'", result.ClusterName)
		}
	})

	t.Run("snapshot not ready", func(t *testing.T) {
		repo := &mockSnapshotRepo{overviewErr: domain.ErrSnapshotNotReady}
		handler := setupTestHandler(repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/overview", nil)
		rec := httptest.NewRecorder()

		handler.GetOverView(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("snapshot not ready")) {
			t.Errorf("expected snapshot not ready error message, got %s", rec.Body.String())
		}
	})

	t.Run("generic internal error", func(t *testing.T) {
		repo := &mockSnapshotRepo{overviewErr: errors.New("db failure")}
		handler := setupTestHandler(repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/overview", nil)
		rec := httptest.NewRecorder()

		handler.GetOverView(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("internal error")) {
			t.Errorf("expected internal error message, got %s", rec.Body.String())
		}
	})
}

func TestHandler_GetNodes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		nodes := []domain.NodeTelemetry{
			{ID: "node-1", Role: "worker", Status: "Ready"},
			{ID: "node-2", Role: "control-plane", Status: "Ready"},
		}
		repo := &mockSnapshotRepo{nodes: nodes}
		handler := setupTestHandler(repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/nodes", nil)
		rec := httptest.NewRecorder()

		handler.GetNodes(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var result []domain.NodeTelemetry
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 nodes, got %d", len(result))
		}
	})

	t.Run("snapshot not ready", func(t *testing.T) {
		repo := &mockSnapshotRepo{nodesErr: domain.ErrSnapshotNotReady}
		handler := setupTestHandler(repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/nodes", nil)
		rec := httptest.NewRecorder()

		handler.GetNodes(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("snapshot not ready")) {
			t.Errorf("expected snapshot not ready error, got %s", rec.Body.String())
		}
	})

	t.Run("generic internal error", func(t *testing.T) {
		repo := &mockSnapshotRepo{nodesErr: errors.New("db failure")}
		handler := setupTestHandler(repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/nodes", nil)
		rec := httptest.NewRecorder()

		handler.GetNodes(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("internal error")) {
			t.Errorf("expected internal error message, got %s", rec.Body.String())
		}
	})
}

func TestHandler_GetServices(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		services := []domain.ServiceTelemetry{
			{Name: "api", Status: domain.StatusHealthy, LatencyMs: 12, LastCheck: time.Now()},
		}
		repo := &mockSnapshotRepo{services: services}
		handler := setupTestHandler(repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/services", nil)
		rec := httptest.NewRecorder()

		handler.GetServices(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var result []domain.ServiceTelemetry
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(result) != 1 || result[0].Name != "api" {
			t.Errorf("unexpected services returned: %v", result)
		}
	})

	t.Run("error", func(t *testing.T) {
		repo := &mockSnapshotRepo{servicesErr: errors.New("connection failed")}
		handler := setupTestHandler(repo)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/services", nil)
		rec := httptest.NewRecorder()

		handler.GetServices(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("internal server error")) {
			t.Errorf("expected internal server error message, got %s", rec.Body.String())
		}
	})
}

func TestHandler_HandleTerminalCommand(t *testing.T) {
	repo := &mockSnapshotRepo{}
	handler := setupTestHandler(repo)

	t.Run("invalid json body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cmd", bytes.NewBufferString("invalid json"))
		rec := httptest.NewRecorder()

		handler.HandleTerminalCommand(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("invalid request body")) {
			t.Errorf("expected invalid request body message, got %s", rec.Body.String())
		}
	})

	t.Run("empty command", func(t *testing.T) {
		payload, _ := json.Marshal(domain.TerminalCommandRequest{Command: ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cmd", bytes.NewBuffer(payload))
		rec := httptest.NewRecorder()

		handler.HandleTerminalCommand(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte("command is required")) {
			t.Errorf("expected command is required message, got %s", rec.Body.String())
		}
	})

	t.Run("unknown command returns 400 with response payload", func(t *testing.T) {
		payload, _ := json.Marshal(domain.TerminalCommandRequest{Command: "invalid-command-xyz"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cmd", bytes.NewBuffer(payload))
		rec := httptest.NewRecorder()

		handler.HandleTerminalCommand(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var resp domain.TerminalCommandResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !bytes.Contains([]byte(resp.OutputText), []byte("Command not found")) {
			t.Errorf("expected command not found output text, got %s", resp.OutputText)
		}
	})

	t.Run("valid command returns 200 with response payload", func(t *testing.T) {
		payload, _ := json.Marshal(domain.TerminalCommandRequest{Command: "help"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cmd", bytes.NewBuffer(payload))
		rec := httptest.NewRecorder()

		handler.HandleTerminalCommand(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		var resp domain.TerminalCommandResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Command != "help" {
			t.Errorf("expected command 'help', got '%s'", resp.Command)
		}
	})
}
