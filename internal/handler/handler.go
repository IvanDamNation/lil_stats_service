package handler

import (
	"encoding/json"
	"log"
	"net/http"

	m "github.com/IvanDamNation/lil_stats_service/internal/models"
)

var (
	ErrInvalidJSON      = "invalid JSON"
	ErrMethodNotAllowed = "method not allowed"

	ErrAutorIdEmpty         = "author id is empty"
	ErrUserIdEmpty          = "user id is empty"
	ErrZeroAuthorsRequested = "zero authors requested"
)

type ClickStorage interface {
	RecordClick(userID m.UserID, authorID m.AuthorID)
	GetUniqueCounts(authorIDs []m.AuthorID) map[m.AuthorID]uint64
}

type Handler struct {
	storage ClickStorage
}

func NewHandler(storage ClickStorage) *Handler {
	return &Handler{storage: storage}
}

type clickRequest struct {
	AuthorId string `json:"author_id"`
	UserId   string `json:"user_id"`
}

type statsRequest struct {
	AuthorIds []string `json:"author_ids"`
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
	h.storage.RecordClick(m.UserID(req.UserId), m.AuthorID(req.AuthorId))

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
	res := h.storage.GetUniqueCounts(toDomainAuthorIDs(req.AuthorIds))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]any{
		"status": http.StatusText(http.StatusOK),
		"stats":  res,
	})
}

func toDomainAuthorIDs(ids []string) []m.AuthorID {
	res := make([]m.AuthorID, len(ids))
	for i, v := range ids {
		res[i] = m.AuthorID(v)
	}
	return res
}
