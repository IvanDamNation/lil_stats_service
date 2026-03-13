package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/IvanDamNation/lil_stats_service/internal/models"
	"github.com/IvanDamNation/lil_stats_service/internal/storage"
)

var (
	ErrInvalidJSON      = "invalid JSON"
	ErrMethodNotAllowed = "method not allowed"

	ErrAutorIdEmpty         = "author id is empty"
	ErrUserIdEmpty          = "user id is empty"
	ErrZeroAuthorsRequested = "zero authors requested"
)

type Handler struct {
	storage storage.Storage
}

type StorageHandler interface {
	Click(w http.ResponseWriter, r *http.Request)
	YesterdayUniqueClicks(w http.ResponseWriter, r *http.Request)
}

func NewHandler(storage storage.Storage) StorageHandler {
	return &Handler{storage: storage}
}

type clickRequest struct {
	AuthorId models.AuthorID `json:"author_id"`
	UserId   models.UserID   `json:"user_id"`
}

type statsRequest struct {
	AuthorIds []models.AuthorID `json:"author_ids"`
}

// POST /api/v1/click
func (h *Handler) Click(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Print("Got wrong method")
		http.Error(w, ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req clickRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("JSON decode error: %v", err)
		http.Error(w, ErrInvalidJSON, http.StatusBadRequest)
		return
	}

	if req.AuthorId == "" {
		log.Print("Got empty author ID")
		http.Error(w, ErrAutorIdEmpty, http.StatusUnprocessableEntity)
		return
	}
	if req.UserId == "" {
		log.Print("Got empty user ID")
		http.Error(w, ErrUserIdEmpty, http.StatusUnprocessableEntity)
		return
	}

	log.Printf("Click got %v \n", req)
	h.storage.RecordClick(req.UserId, req.AuthorId)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  http.StatusText(http.StatusAccepted),
		"message": "click processed",
	})
}

// POST /api/v1/stats
func (h *Handler) YesterdayUniqueClicks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Print("Got wrong method")
		http.Error(w, ErrMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req statsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("JSON decode error: %v", err)
		http.Error(w, ErrInvalidJSON, http.StatusBadRequest)
		return
	}

	if len(req.AuthorIds) == 0 {
		log.Printf("Got empty author list")
		http.Error(w, ErrZeroAuthorsRequested, http.StatusUnprocessableEntity)
		return
	}

	log.Printf("Got author list with length: %d", len(req.AuthorIds))
	res := h.storage.GetUniqueCounts(req.AuthorIds)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]any{
		"status": http.StatusText(http.StatusOK),
		"stats":  res,
	})
}
