package container

import (
	"sync"
)

const MapTSDefaultCapacity = 256

type MapTS[K comparable, V any] struct {
	m  map[K]V
	mu sync.RWMutex
}

func NewMapTS[K comparable, V any](capacity ...int) *MapTS[K, V] {
	cap := MapTSDefaultCapacity

	if len(capacity) > 0 && capacity[0] > 0 {
		cap = capacity[0]
	}

	return &MapTS[K, V]{
		m: make(map[K]V, cap),
	}
}

// Set adds or updates a key-value pair with a TTL
func (m *MapTS[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m[key] = value
}

// GetOK retrieves a value by key if it exists and hasn't expired
func (m *MapTS[K, V]) GetOK(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.m[key]
	if !exists {
		var zero V
		return zero, false
	}

	return item, true
}

func (m *MapTS[K, V]) Get(key K) V {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.m[key]
}

func (m *MapTS[K, V]) Has(key K) bool {
	_, exists := m.GetOK(key)

	return exists
}

// Delete removes an item from the cache
func (m *MapTS[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.m, key)
}

// Clear removes all items from the cache
func (m *MapTS[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m = make(map[K]V, MapTSDefaultCapacity)
}

// Size returns the number of items in the shard
func (m *MapTS[K, V]) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.m)
}

func (m *MapTS[K, V]) MapUnsafe() map[K]V {
	return m.m
}

func (m *MapTS[K, V]) Lock() {
	m.mu.Lock()
}

func (m *MapTS[K, V]) Unlock() {
	m.mu.Unlock()
}

func (m *MapTS[K, V]) RLock() {
	m.mu.RLock()
}

func (m *MapTS[K, V]) RUnlock() {
	m.mu.RUnlock()
}
