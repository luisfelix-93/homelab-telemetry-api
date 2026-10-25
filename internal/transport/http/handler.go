package transporthttp

import (
	"encoding/json"
	"errors"
	"homelab-telemetry-api/internal/domain"
	"homelab-telemetry-api/internal/service"
	"net/http"
)

type Handler struct {
	repo          domain.SnapshotRepository
	commandParser *service.CommandParser
}

func NewHandler(repo domain.SnapshotRepository, cmdParser *service.CommandParser) *Handler {
	return &Handler{
		repo:          repo,
		commandParser: cmdParser,
	}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetOverView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	overview, err := h.repo.GetOverView(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrSnapshotNotReady) {
			http.Error(w, `{"error": "snapshot not ready"}`, http.StatusInternalServerError)
			return
		}
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(overview)
}

func (h *Handler) GetNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodes, err := h.repo.GetNodes(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrSnapshotNotReady) {
			http.Error(w, `{"error": "snapshot not ready"}`, http.StatusInternalServerError)
			return
		}
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(nodes)
}

func (h *Handler) GetServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	services, err := h.repo.GetServices(ctx)
	if err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(services)
}

func (h *Handler) HandleTerminalCommand(w http.ResponseWriter, r *http.Request) {
	var req domain.TerminalCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, `{"error": "command is required"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	resp, err := h.commandParser.Execute(ctx, req)
	if err != nil && errors.Is(err, domain.ErrCommandNotFound) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}