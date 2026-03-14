package storage

import (
	"log"
	"sync"
	"time"

	"github.com/IvanDamNation/lil_stats_service/internal/models"
)

type userID = models.UserID
type authorID = models.AuthorID

type Storage interface {
	RecordClick(userID, authorID)
	GetUniqueCounts(authorIDs []authorID) map[authorID]uint64

	Stop()
}

type countStorage struct {
	today     map[authorID]map[userID]struct{}
	yesterday map[authorID]uint64
	mu        sync.RWMutex

	done chan struct{}
	stop chan struct{}

	onRotate func() // for tests
}

func NewStorage(timeProvider func() time.Duration) Storage {
	storage := &countStorage{
		today:     make(map[authorID]map[userID]struct{}),
		yesterday: make(map[authorID]uint64),

		done: make(chan struct{}),
		stop: make(chan struct{}),
	}

	go storage.rotateLoop(timeProvider)

	return storage
}

func (cs *countStorage) RecordClick(u userID, a authorID) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.today[a]; !exists {
		cs.today[a] = make(map[userID]struct{})
	}
	cs.today[a][u] = struct{}{}
}

func (cs *countStorage) GetUniqueCounts(authorIDs []authorID) map[authorID]uint64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	stats := make(map[authorID]uint64, len(authorIDs))

	for _, author := range authorIDs {
		count := cs.yesterday[author]
		stats[author] = count
	}

	return stats
}

func (cs *countStorage) rotateLoop(timeProvider func() time.Duration) {
	defer func() {
		log.Print("storage worker stopped")
		close(cs.done)
	}()

	for {
		dur := timeProvider()

		log.Printf("rotation scheduled to: %v\n", dur)
		timer := time.NewTimer(dur)

		select {
		case <-timer.C:
			cs.rotate()
		case <-cs.stop:
			timer.Stop()
			return
		}
	}
}

func (cs *countStorage) rotate() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	newYesterday := make(map[authorID]uint64, len(cs.today))
	for author, userSet := range cs.today {
		newYesterday[author] = uint64(len(userSet))
	}
	cs.yesterday = newYesterday

	cs.today = make(map[authorID]map[userID]struct{})

	if cs.onRotate != nil {
		cs.onRotate()
	}
}

func (cs *countStorage) Stop() {
	close(cs.stop)
	<-cs.done
}

func NowFunc() time.Duration {
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return nextMidnight.Sub(now)
}
