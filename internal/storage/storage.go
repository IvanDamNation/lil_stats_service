package storage

import (
	"log"
	"sync"
	"time"

	m "github.com/IvanDamNation/lil_stats_service/internal/models"
)

type countStorage struct {
	today     map[m.AuthorID]map[m.UserID]struct{}
	yesterday map[m.AuthorID]uint64
	mu        sync.RWMutex

	done chan struct{}
	stop chan struct{}
}

func NewStorage(timeProvider func() time.Duration) *countStorage {
	storage := &countStorage{
		today:     make(map[m.AuthorID]map[m.UserID]struct{}),
		yesterday: make(map[m.AuthorID]uint64),

		done: make(chan struct{}),
		stop: make(chan struct{}),
	}
	ticks := make(chan time.Time)

	go func() {
		defer close(ticks)

		for {
			dur := timeProvider()
			timer := time.NewTimer(dur)
			select {
			case t := <-timer.C:
				select {
				case ticks <- t:
				case <-storage.stop:
					timer.Stop()
					return
				}
			case <-storage.stop:
				timer.Stop()
				return
			}
		}
	}()

	go storage.rotateLoop(ticks)

	return storage
}

func (cs *countStorage) RecordClick(u m.UserID, a m.AuthorID) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.today[a]; !exists {
		cs.today[a] = make(map[m.UserID]struct{})
	}
	cs.today[a][u] = struct{}{}
}

func (cs *countStorage) GetUniqueCounts(authorIDs []m.AuthorID) map[m.AuthorID]uint64 {
	stats := make(map[m.AuthorID]uint64, len(authorIDs))

	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, author := range authorIDs {
		count := cs.yesterday[author]
		stats[author] = count
	}

	return stats
}

func (cs *countStorage) rotateLoop(nextTick <-chan time.Time) {
	defer func() {
		log.Print("storage worker stopped")
		close(cs.done)
	}()

	for {
		select {
		case <-nextTick:
			cs.rotate()
		case <-cs.stop:
			return
		}
	}
}

func (cs *countStorage) rotate() {
	newToday := make(map[m.AuthorID]map[m.UserID]struct{})

	cs.mu.Lock()
	oldToday := cs.today
	cs.today = newToday
	cs.mu.Unlock()

	newYesterday := make(map[m.AuthorID]uint64, len(oldToday))
	for author, userSet := range oldToday {
		newYesterday[author] = uint64(len(userSet))
	}

	cs.mu.Lock()
	cs.yesterday = newYesterday
	cs.mu.Unlock()
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
