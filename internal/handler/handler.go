package handler

import (
	"net/http"

	"github.com/IvanDamNation/lil_stats_service/internal/storage"
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

// TODO
func (h *Handler) Click(w http.ResponseWriter, r *http.Request)                 {}
// TODO
func (h *Handler) YesterdayUniqueClicks(w http.ResponseWriter, r *http.Request) {}
