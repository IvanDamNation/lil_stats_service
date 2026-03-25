package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	m "github.com/IvanDamNation/lil_stats_service/internal/models"
)

// Implements ClickStorage for tests
type mockStorage struct {
	recordedClicks []struct {
		userID   m.UserID
		authorID m.AuthorID
	}
	returnCounts        map[m.AuthorID]uint64
	getUniqueCountsArgs []m.AuthorID

	getLastEventsAmount int
	returnEvents        []m.ClickEvent
	returnEventsErr     error
}

// Mocking ClickStorage method for registering clicks
func (ms *mockStorage) RecordClick(userID m.UserID, authorID m.AuthorID) {
	ms.recordedClicks = append(ms.recordedClicks, struct {
		userID   m.UserID
		authorID m.AuthorID
	}{userID, authorID})
}

// Mocking ClickStorage method for yesterday stats
func (ms *mockStorage) GetUniqueCounts(authorIDs []m.AuthorID) map[m.AuthorID]uint64 {
	ms.getUniqueCountsArgs = authorIDs
	result := make(map[m.AuthorID]uint64, len(authorIDs))
	for _, a := range authorIDs {
		if count, ok := ms.returnCounts[a]; ok {
			result[a] = count
		} else {
			result[a] = 0
		}
	}

	return result
}

// Mocking ClickStorage method for latest events in ring buffer
func (ms *mockStorage) GetLastEvents(amount int) ([]m.ClickEvent, error) {
	ms.getLastEventsAmount = amount
	return ms.returnEvents, ms.returnEventsErr
}

// Test for Click
func TestClickHandler(t *testing.T) {
	mock := &mockStorage{}
	h := NewHandler(mock)

	reqBody := clickRequest{
		AuthorId: "author123",
		UserId:   "user456",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/click", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Click(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected status Accepted(202), got %v", resp.StatusCode)
	}
	if mock.recordedClicks[0].userID != "user456" {
		t.Errorf("expected userID user456, got %v", mock.recordedClicks[0].userID)
	}
	if mock.recordedClicks[0].authorID != "author123" {
		t.Errorf("expected authorID author123, got %v", mock.recordedClicks[0].authorID)
	}
}

// Test for YesterdayUniqueClicks
func TestYesterdayUniqueClicksHandler(t *testing.T) {
	mock := &mockStorage{
		returnCounts: map[m.AuthorID]uint64{
			"author1": 5,
			"author2": 3,
		},
	}
	h := NewHandler(mock)
	reqBody := statsRequest{
		AuthorIds: []string{"author1", "author2", "author3"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost, "/api/v1/click_stats", bytes.NewReader(body),
	)
	w := httptest.NewRecorder()
	h.YesterdayUniqueClicks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected OK(200), got %v", w.Code)
	}
	if len(mock.getUniqueCountsArgs) != 3 {
		t.Fatalf("expected 3 authors in args, got %v", mock.getUniqueCountsArgs)
	}
	if mock.getUniqueCountsArgs[0] != "author1" ||
		mock.getUniqueCountsArgs[1] != "author2" ||
		mock.getUniqueCountsArgs[2] != "author3" {
		t.Errorf("unexpected args: %v", mock.getUniqueCountsArgs)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	stats, ok := resp["stats"].(map[string]any)
	if !ok {
		t.Fatal("field not found in map")
	}
	if stats["author1"] != float64(5) {
		t.Errorf("author1 expected 5, got %v", stats["author1"])
	}
	if stats["author2"] != float64(3) {
		t.Errorf("author2 expected 3, got %v", stats["author2"])
	}
	if stats["author3"] != float64(0) {
		t.Errorf("author3 expected 0, got %v", stats["author3"])
	}
}

// Test for LastEvents
func TestGetEventsHandler(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockEvents     []m.ClickEvent
		mockErr        error
		expectedStatus int
		expectedBody   []clickEventsResponse
	}{
		{
			name: "success",
			url:  "/api/v1/last_events?amount=10",
			mockEvents: []m.ClickEvent{
				{Author: "author1", User: "user1"},
				{Author: "author2", User: "user2"}},
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedBody: []clickEventsResponse{
				{AuthorID: "author1", UserID: "user1"},
				{AuthorID: "author2", UserID: "user2"}},
		},
		{
			name:           "storage error",
			url:            "/api/v1/last_events?amount=10",
			mockEvents:     nil,
			mockErr:        errors.New("mock error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
		{
			name:           "zero amount",
			url:            "/api/v1/last_events?amount=0",
			expectedStatus: http.StatusOK,
			expectedBody:   []clickEventsResponse{},
		},
		{
			name:           "missing amount",
			url:            "/api/v1/last_events",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid amount",
			url:            "/api/v1/last_events?amount=asdf",
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "negative amount",
			url:            "/api/v1/last_events?amount=-2",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		mock := &mockStorage{
			returnEvents:    tt.mockEvents,
			returnEventsErr: tt.mockErr,
		}
		h := NewHandler(mock)

		req := httptest.NewRequest(http.MethodGet, tt.url, nil)
		w := httptest.NewRecorder()

		h.LastEvents(w, req)
		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != tt.expectedStatus {
			t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
		}

		if tt.expectedBody != nil {
			var body []clickEventsResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if !reflect.DeepEqual(body, tt.expectedBody) {
				t.Errorf("expected body %v, got %v", tt.expectedBody, body)
			}
		}

		if tt.expectedStatus == http.StatusOK && tt.mockErr == nil {
			var expectedAmount int
			if tt.name != "zero amount" {
				expectedAmount = 10
			}
			if mock.getLastEventsAmount != expectedAmount {
				t.Errorf("expected amount %d, got %d", expectedAmount, mock.getLastEventsAmount)
			}
		}
	}
}
