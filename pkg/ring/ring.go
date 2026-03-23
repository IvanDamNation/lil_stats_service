// package contains generic ring/circular buffer implementation
package ring

import (
	"errors"
	"sync"
)

var (
	ErrBufferCapacityInput = errors.New("ring buffer: init capacity must be >= 1")
	ErrEmptyBuffer         = errors.New("ring buffer: empty")
	ErrNegativeAmountInput = errors.New("ring buffer: amount must be >= 0")
)

type ringBuffer[T any] struct {
	buf  []T
	size int

	readIdx  int
	writeIdx int

	mu sync.Mutex
}

func NewRingBuffer[T any](capacity int) (*ringBuffer[T], error) {
	if capacity <= 0 {
		return nil, ErrBufferCapacityInput
	}

	rb := &ringBuffer[T]{
		buf: make([]T, capacity),
	}
	return rb, nil
}

func (rb *ringBuffer[T]) Push(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.size == len(rb.buf) {
		rb.readIdx = (rb.readIdx + 1) % len(rb.buf)
	} else {
		rb.size++
	}

	rb.buf[rb.writeIdx] = item
	rb.writeIdx = (rb.writeIdx + 1) % len(rb.buf)
}

func (rb *ringBuffer[T]) Pop() (T, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.size == 0 {
		var defaultVal T
		return defaultVal, ErrEmptyBuffer
	}

	item := rb.buf[rb.readIdx]
	rb.readIdx = (rb.readIdx + 1) % len(rb.buf)
	rb.size--

	return item, nil
}

func (rb *ringBuffer[T]) Peek() (T, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.size == 0 {
		var defaultVal T
		return defaultVal, ErrEmptyBuffer
	}
	return rb.buf[rb.readIdx], nil
}

func (rb *ringBuffer[T]) GetLast(amount int) ([]T, error) {
	if amount < 0 {
		return nil, ErrNegativeAmountInput
	}
	if amount == 0 {
		return []T{}, nil
	}

	items := make([]T, 0, amount)

	rb.mu.Lock()
	if amount > rb.size {
		amount = rb.size
	}

	curr := (rb.writeIdx - 1 + len(rb.buf)) % len(rb.buf)
	for range amount {
		items = append(items, rb.buf[curr])
		curr = (curr - 1 + len(rb.buf)) % len(rb.buf)
	}
	rb.mu.Unlock()

	return items, nil
}

func (rb *ringBuffer[T]) Size() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	return rb.size
}

func (rb *ringBuffer[T]) Capacity() int {
	return len(rb.buf)
}

func (rb *ringBuffer[T]) IsFull() bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	return rb.size == len(rb.buf)
}

func (rb *ringBuffer[T]) IsEmpty() bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	return rb.size == 0
}

func (rb *ringBuffer[T]) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	clear(rb.buf)
	rb.size, rb.writeIdx, rb.readIdx = 0, 0, 0
}
