package storage

import (
	"sync"
	"testing"
	"time"

	m "github.com/IvanDamNation/lil_stats_service/internal/models"
)

func TestClickAndGetUnique(t *testing.T) {
	s := &countStorage{
		today:     make(map[m.AuthorID]map[m.UserID]struct{}),
		yesterday: make(map[m.AuthorID]uint64),
	}

	author1 := m.AuthorID("author1")
	author2 := m.AuthorID("author2")
	user1 := m.UserID("user1")
	user2 := m.UserID("user2")

	s.RecordClick(user1, author1)
	s.RecordClick(user2, author1)
	s.RecordClick(user1, author2)

	res := s.GetUniqueCounts([]m.AuthorID{author1, author2, "author3"})
	if len(res) != 3 {
		t.Errorf("expected 3, got %d", len(res))
	}
	if res[author1] != 0 {
		t.Errorf("expected 0 for author 1, got %d", res[author1])
	}
	if res[author2] != 0 {
		t.Errorf("expected 0 for author 2, got %d", res[author2])
	}
	if res["author3"] != 0 {
		t.Errorf("expected 0 for author 3, got %d", res["author3"])
	}

	s.mu.RLock()
	if len(s.today) != 2 {
		t.Errorf("expected 2 in today, got %d", len(res))
	}
	if len(s.today[author1]) != 2 {
		t.Errorf("expected 2 users for author1, got %d", len(s.today[author1]))
	}
	if len(s.today[author2]) != 1 {
		t.Errorf("expected 1 user for author2, got %d", len(s.today[author2]))
	}
	s.mu.RUnlock()
}

func TestRotate(t *testing.T) {
	s := &countStorage{
		today:     make(map[m.AuthorID]map[m.UserID]struct{}),
		yesterday: make(map[m.AuthorID]uint64),
	}

	author1 := m.AuthorID("author1")
	author2 := m.AuthorID("author2")
	user1 := m.UserID("user1")
	user2 := m.UserID("user2")
	user3 := m.UserID("user3")

	s.RecordClick(user1, author1)
	s.RecordClick(user2, author1)
	s.RecordClick(user3, author1)
	s.RecordClick(user1, author2)
	s.RecordClick(user1, author2)

	s.rotate()

	s.mu.RLock()
	if len(s.today) != 0 {
		t.Errorf("today should be empty after rotate, got %d", len(s.today))
	}
	if len(s.yesterday) != 2 {
		t.Errorf("yesterday expected 2 authors, got %d", len(s.yesterday))
	}
	if s.yesterday[author1] != 3 {
		t.Errorf("yesterday[author1] expected 3, got %d", s.yesterday[author1])
	}
	if s.yesterday[author2] != 1 {
		t.Errorf("yesterday[author2] expected 1, got %d", s.yesterday[author2])
	}
	s.mu.RUnlock()

	res := s.GetUniqueCounts([]m.AuthorID{author1, author2})
	if res[author1] != 3 {
		t.Errorf("GetUniqueCounts after rotate: author1 expected 3, got %d", res[author1])
	}
	if res[author2] != 1 {
		t.Errorf("GetUniqueCounts after rotate: author2 expected 1, got %d", res[author2])
	}

	s.RecordClick(user1, author1)
	res = s.GetUniqueCounts([]m.AuthorID{author1})
	if res[author1] != 3 {
		t.Errorf("after new click, yesterday still 3, got %d", res[author1])
	}
	s.rotate()
	res = s.GetUniqueCounts([]m.AuthorID{author1})
	if res[author1] != 1 {
		t.Errorf("after 2nd rotate, author1 expected 1, got %d", res[author1])
	}
}

func TestConcurrent(t *testing.T) {
	s := &countStorage{
		today:     make(map[m.AuthorID]map[m.UserID]struct{}),
		yesterday: make(map[m.AuthorID]uint64),
	}

	const authors = 10
	const users = 100
	const clickPerUser = 5

	var wg sync.WaitGroup
	for a := range authors {
		author := m.AuthorID(rune('A' + a))
		for u := range users {
			user := m.UserID(rune('u' + u))
			wg.Add(1)
			go func(author m.AuthorID, user m.UserID) {
				defer wg.Done()
				for range clickPerUser {
					s.RecordClick(user, author)
				}
			}(author, user)
		}
	}
	wg.Wait()

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.today) != authors {
		t.Errorf("expected %d authors, got %d", authors, len(s.today))
	}
	for a := range authors {
		author := m.AuthorID(rune('A' + a))
		if len(s.today[author]) != users {
			t.Errorf(
				"author %v expected %d users, got %d",
				author, users, len(s.today[author]))
		}
	}
}

func TestRotateLoop(t *testing.T) {
	ticks := make(chan time.Time)

	s := &countStorage{
		today:     make(map[m.AuthorID]map[m.UserID]struct{}),
		yesterday: make(map[m.AuthorID]uint64),
		done:      make(chan struct{}),
	}

	authorID := m.AuthorID("test-author")
	s.today[authorID] = map[m.UserID]struct{}{"user1": {}}

	go s.rotateLoop(ticks)

	ticks <- time.Now()

	timeout := time.After(100 * time.Millisecond)
	tickProcessed := make(chan bool)
	go func() {
		for {
			s.mu.RLock()
			count := s.yesterday[authorID]
			s.mu.RUnlock()

			if count == 1 {
				tickProcessed <- true
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-tickProcessed: // ok
	case <-timeout:
		t.Fatal("rotate not called within timeout")
	}
}
