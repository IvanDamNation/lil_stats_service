package storage

import (
	"context"
	"log"
	"sync"
	"time"

	m "github.com/IvanDamNation/lil_stats_service/internal/models"
	"github.com/IvanDamNation/lil_stats_service/pkg/ring"
)

type clickEventsRing[T any] interface {
	Push(T)
	GetLast(int) ([]T, error)
}

type countStorage struct {
	today     map[m.AuthorID]map[m.UserID]struct{}
	yesterday map[m.AuthorID]uint64
	ringBuf   clickEventsRing[m.ClickEvent]
	mu        sync.RWMutex

	done chan struct{}
}

func NewStorage(ctx context.Context, capacity int, timeProvider func() time.Duration) (*countStorage, error) {
	ringBuf, err := ring.NewRingBuffer[m.ClickEvent](capacity)
	if err != nil {
		return nil, err
	}

	storage := &countStorage{
		today:     make(map[m.AuthorID]map[m.UserID]struct{}),
		yesterday: make(map[m.AuthorID]uint64),
		ringBuf:   ringBuf,

		done: make(chan struct{}),
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
				case <-ctx.Done():
					timer.Stop()
					return
				}
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()

	go storage.rotateLoop(ticks)

	return storage, nil
}

func (cs *countStorage) RecordClick(u m.UserID, a m.AuthorID) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.ringBuf.Push(m.ClickEvent{Author: a, User: u})
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

func (cs *countStorage) GetLastEvents(amount int) ([]m.ClickEvent, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	items, err := cs.ringBuf.GetLast(amount)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (cs *countStorage) rotateLoop(nextTick <-chan time.Time) {
	defer func() {
		log.Print("storage worker stopped")
		close(cs.done)
	}()

	for range nextTick {
		cs.rotate()
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

func (cs *countStorage) Wait() {
	<-cs.done
}

func NowFunc() time.Duration {
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return nextMidnight.Sub(now)
}
