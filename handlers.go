package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// HTTPHandlers provides HTTP handlers for the REST API
type HTTPHandlers struct {
	pubsub *PubSubSystem
}

// NewHTTPHandlers creates a new HTTP handlers instance
func NewHTTPHandlers(pubsub *PubSubSystem) *HTTPHandlers {
	return &HTTPHandlers{pubsub: pubsub}
}

// CreateTopic handles POST /topics
func (h *HTTPHandlers) CreateTopic(w http.ResponseWriter, r *http.Request) {
	var req CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Topic name is required", http.StatusBadRequest)
		return
	}

	err := h.pubsub.CreateTopic(req.Name)
	if err != nil {
		// Topic already exists
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)

		resp := CreateTopicResponse{
			Status: "exists",
			Topic:  req.Name,
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Topic created successfully
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := CreateTopicResponse{
		Status: "created",
		Topic:  req.Name,
	}
	json.NewEncoder(w).Encode(resp)
}

// DeleteTopic handles DELETE /topics/{name}
func (h *HTTPHandlers) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topicName := vars["name"]

	if topicName == "" {
		http.Error(w, "Topic name is required", http.StatusBadRequest)
		return
	}

	err := h.pubsub.DeleteTopic(topicName)
	if err != nil {
		// Topic not found
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		errorResp := map[string]string{
			"error": "Topic not found",
		}
		json.NewEncoder(w).Encode(errorResp)
		return
	}

	// Topic deleted successfully
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := DeleteTopicResponse{
		Status: "deleted",
		Topic:  topicName,
	}
	json.NewEncoder(w).Encode(resp)
}

// GetTopics handles GET /topics
func (h *HTTPHandlers) GetTopics(w http.ResponseWriter, r *http.Request) {
	topics := h.pubsub.GetTopics()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := TopicsResponse{
		Topics: topics,
	}
	json.NewEncoder(w).Encode(resp)
}

// GetHealth handles GET /health
func (h *HTTPHandlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	health := h.pubsub.GetHealth()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(health)
}

// GetStats handles GET /stats
func (h *HTTPHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.pubsub.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(stats)
}

// GetSubscriptionsStatus handles GET /subscriptions
func (h *HTTPHandlers) GetSubscriptionsStatus(w http.ResponseWriter, r *http.Request) {
	status := h.pubsub.GetSubscriptionsStatus()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(status)
}

// SetupRoutes configures the HTTP routes
func (h *HTTPHandlers) SetupRoutes(router *mux.Router) {
	// Topic management
	router.HandleFunc("/topics", h.CreateTopic).Methods("POST")
	router.HandleFunc("/topics/{name}", h.DeleteTopic).Methods("DELETE")
	router.HandleFunc("/topics", h.GetTopics).Methods("GET")

	// System endpoints
	router.HandleFunc("/health", h.GetHealth).Methods("GET")
	router.HandleFunc("/stats", h.GetStats).Methods("GET")
	router.HandleFunc("/subscriptions", h.GetSubscriptionsStatus).Methods("GET")

	// WebSocket endpoint
	router.HandleFunc("/ws", HandleWebSocket(h.pubsub)).Methods("GET")
}
