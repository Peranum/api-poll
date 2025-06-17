package container

import (
	"sync"
	"testing"
)

func TestMapTS_SetGet(t *testing.T) {
	m := NewMapTS[string, int]()
	m.Set("key1", 10)
	value, exists := m.GetOK("key1")
	if !exists || value != 10 {
		t.Errorf("Expected 10, got %v", value)
	}
}

func TestMapTS_Delete(t *testing.T) {
	m := NewMapTS[string, int]()
	m.Set("key1", 10)
	m.Delete("key1")
	_, exists := m.GetOK("key1")
	if exists {
		t.Error("Expected key1 to be deleted")
	}
}

func TestMapTS_ConcurrentAccess(t *testing.T) {
	m := NewMapTS[string, int]()
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m.Set("key", i)
		}(i)
	}
	wg.Wait()
	// Check if the last set value is present
	value, exists := m.GetOK("key")
	if !exists {
		t.Error("Expected key to exist")
	}
	if value < 0 || value >= 1000 {
		t.Errorf("Unexpected value: %v", value)
	}
}

func TestShardedMapTS_SetGet(t *testing.T) {
	m := NewShardedMapTS[string, int]()
	m.Set("key1", 10)
	value, exists := m.GetOK("key1")
	if !exists || value != 10 {
		t.Errorf("Expected 10, got %v", value)
	}
}

func TestShardedMapTS_ConcurrentAccess(t *testing.T) {
	m := NewShardedMapTS[string, int]()
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m.Set("key", i)
		}(i)
	}
	wg.Wait()
	// Check if the last set value is present
	value, exists := m.GetOK("key")
	if !exists {
		t.Error("Expected key to exist")
	}
	if value < 0 || value >= 1000 {
		t.Errorf("Unexpected value: %v", value)
	}
}
