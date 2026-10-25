package transporthttp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"homelab-telemetry-api/internal/domain"
	transporthttp "homelab-telemetry-api/internal/transport/http"
)

func TestRouter_Integration(t *testing.T) {
	repo := &mockSnapshotRepo{
		overview: &domain.ClusterOverview{
			ClusterName: "homelab-router-test",
		},
		services: []domain.ServiceTelemetry{
			{Name: "api-test", Status: domain.StatusHealthy, LastCheck: time.Now()},
		},
		nodes: []domain.NodeTelemetry{
			{ID: "node-test", Role: "worker", Status: "Ready"},
		},
	}
	handler := setupTestHandler(repo)
	router := transporthttp.NewRouter(handler)

	t.Run("CORS preflight request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200 for OPTIONS, got %d", rec.Code)
		}
		if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
		}
		if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, OPTIONS" {
			t.Errorf("expected Access-Control-Allow-Methods 'GET, POST, OPTIONS', got '%s'", methods)
		}
		if headers := rec.Header().Get("Access-Control-Allow-Headers"); headers != "Content-Type, Authorization" {
			t.Errorf("expected Access-Control-Allow-Headers 'Content-Type, Authorization', got '%s'", headers)
		}
	})

	t.Run("GET /api/v1/health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("expected CORS header on GET, got '%s'", origin)
		}
	})

	t.Run("POST /api/v1/cmd", func(t *testing.T) {
		payload, _ := json.Marshal(domain.TerminalCommandRequest{Command: "help"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cmd", bytes.NewBuffer(payload))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/telemetry/overview", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/overview", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/telemetry/services", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/services", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/telemetry/nodes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/nodes", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/non-existent-route returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/non-existent-route", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}
