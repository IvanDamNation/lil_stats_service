package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	m "github.com/IvanDamNation/lil_stats_service/internal/models"
)

var (
	ErrInvalidJSON      = errors.New("invalid JSON")
	ErrJSONEncode       = errors.New("JSON encoding error")
	ErrMethodNotAllowed = errors.New("method not allowed")

	ErrAmountNegativeParam   = errors.New("amount query parameter is negative number")
	ErrAmountQueryNAN        = errors.New("amount query is not a number")
	ErrAmountQueryEmptyParam = errors.New("amount query parameter is not provided")

	ErrAuthorIdEmpty        = errors.New("author id is empty")
	ErrUserIdEmpty          = errors.New("user id is empty")
	ErrZeroAuthorsRequested = errors.New("zero authors requested")
)

type ClickStorage interface {
	RecordClick(userID m.UserID, authorID m.AuthorID)
	GetLastEvents(amount int) ([]m.ClickEvent, error)
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

type clickEventsResponse struct {
	AuthorID string `json:"author_id"`
	UserID   string `json:"user_id"`
}

// GET /api/v1/last_events?amount={int}
func (h *Handler) LastEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Print("Got wrong method")
		http.Error(w, ErrMethodNotAllowed.Error(), http.StatusMethodNotAllowed)
		return
	}

	rawAmount := r.URL.Query().Get("amount")
	if rawAmount == "" {
		log.Print("Empty amount request")
		http.Error(w, ErrAmountQueryEmptyParam.Error(), http.StatusBadRequest)
		return
	}

	amount, err := strconv.Atoi(rawAmount)
	if err != nil {
		log.Printf("Invalid amount format: %s", rawAmount)
		http.Error(w, ErrAmountQueryNAN.Error(), http.StatusUnprocessableEntity)
		return
	}

	if amount < 0 {
		log.Printf("Negative amount: %d", amount)
		http.Error(w, ErrAmountNegativeParam.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Get request last events: %d", amount)
	events, err := h.storage.GetLastEvents(amount)
	if err != nil {
		log.Printf("Storage error: %v", err)
		http.Error(w, "failed to retrieve events", http.StatusInternalServerError)
		return
	}

	res := make([]clickEventsResponse, len(events))
	for i, v := range events {
		res[i] = clickEventsResponse{
			AuthorID: string(v.Author),
			UserID:   string(v.User),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(&res)
	if err != nil {
		log.Print("LastEvents: JSON encode error")
		http.Error(w, ErrJSONEncode.Error(), http.StatusInternalServerError)
		return
	}
}

// POST /api/v1/click
func (h *Handler) Click(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Print("Got wrong method")
		http.Error(w, ErrMethodNotAllowed.Error(), http.StatusMethodNotAllowed)
		return
	}

	var req clickRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("JSON decode error: %v", err)
		http.Error(w, ErrInvalidJSON.Error(), http.StatusBadRequest)
		return
	}

	if req.AuthorId == "" {
		log.Print("Got empty author ID")
		http.Error(w, ErrAuthorIdEmpty.Error(), http.StatusUnprocessableEntity)
		return
	}
	if req.UserId == "" {
		log.Print("Got empty user ID")
		http.Error(w, ErrUserIdEmpty.Error(), http.StatusUnprocessableEntity)
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
		http.Error(w, ErrMethodNotAllowed.Error(), http.StatusMethodNotAllowed)
		return
	}

	var req statsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("JSON decode error: %v", err)
		http.Error(w, ErrInvalidJSON.Error(), http.StatusBadRequest)
		return
	}

	if len(req.AuthorIds) == 0 {
		log.Printf("Got empty author list")
		http.Error(w, ErrZeroAuthorsRequested.Error(), http.StatusUnprocessableEntity)
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
