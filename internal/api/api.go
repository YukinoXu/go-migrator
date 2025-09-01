package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"example.com/go-migrator/internal/models"
	"example.com/go-migrator/internal/queue"
	"example.com/go-migrator/internal/store"
)

type Handler struct {
	store store.Store
	q     queue.Client
	mux   *http.ServeMux
}

// NewHandler creates an API handler. q may be nil; if provided, created task IDs
// will be published to the queue.
func NewHandler(s store.Store, q queue.Client) *Handler {
	h := &Handler{store: s, q: q, mux: http.NewServeMux()}
	h.routes()
	return h
}

func (h *Handler) Router() http.Handler { return h.mux }

func (h *Handler) routes() {
	h.mux.HandleFunc("/tasks", h.tasks)
	h.mux.HandleFunc("/tasks/", h.taskByID)
}

func (h *Handler) tasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var in struct {
			Source  string            `json:"source"`
			Target  string            `json:"target"`
			Payload map[string]string `json:"payload"`
		}
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &in); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		t := &models.Task{Source: in.Source, Target: in.Target, Payload: in.Payload}
		id, err := h.store.CreateTask(t)
		if err != nil {
			http.Error(w, "create error", http.StatusInternalServerError)
			return
		}
		// publish to queue if available
		if h.q != nil {
			if err := h.q.Publish(r.Context(), id); err != nil {
				log.Printf("warning: failed to publish task %s to queue: %v", id, err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
	case http.MethodGet:
		list, _ := h.store.ListTasks()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) taskByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/tasks/")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	t, err := h.store.GetTask(id)
	if err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("store error: %v", err)
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}
