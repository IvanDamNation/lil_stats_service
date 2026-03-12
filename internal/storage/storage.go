package storage

import (
	"sync"

	"github.com/IvanDamNation/lil_stats_service/internal/models"
)

type userID = models.UserID
type authorID = models.AuthorID

type Storage interface {
	RecordClick(userID, authorID)
	GetUniqueCounts(authorIDs []authorID) map[authorID]uint64
}

type countStorage struct {
	today     map[authorID]map[userID]struct{}
	yesterday map[authorID]uint64
	mu        sync.RWMutex
}

func NewStorage() Storage {
	return &countStorage{
		today:     make(map[authorID]map[userID]struct{}),
		yesterday: make(map[authorID]uint64),
	}
}

// TODO
func (cs *countStorage) RecordClick(u userID, a authorID) {

}

// TODO
func (cs *countStorage) GetUniqueCounts(authorIDs []authorID) map[authorID]uint64 {
	return make(map[authorID]uint64)
}
