package container

import (
	"fmt"
	"sync"
	"testing"
)

func BenchmarkMapTS_SyncSet(b *testing.B) {
	m := NewMapTS[string, int]()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		m.Set(key, i)
	}
}

func BenchmarkShardedMapTS_SyncSet(b *testing.B) {
	m := NewShardedMapTS[string, int]()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		m.Set(key, i)
	}
}

func BenchmarkMapTS_SyncGet(b *testing.B) {
	m := NewMapTS[string, int]()
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i)
		m.Set(key, i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%10000)
		m.Get(key)
	}
}

func BenchmarkShardedMapTS_SyncGet(b *testing.B) {
	m := NewShardedMapTS[string, int]()
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i)
		m.Set(key, i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%10000)
		m.Get(key)
	}
}

func BenchmarkMapTS_AsyncSet(b *testing.B) {
	m := NewMapTS[string, int]()

	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			m.Set(key, i)
		}(i)
	}

	wg.Wait()
}

func BenchmarkShardedMapTS_AsyncSet(b *testing.B) {
	m := NewShardedMapTS[string, int]()

	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			m.Set(key, i)
		}(i)
	}

	wg.Wait()
}

func BenchmarkMapTS_AsyncGet(b *testing.B) {
	m := NewMapTS[string, int]()

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i)
		m.Set(key, i)
	}

	b.ResetTimer()

	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i%10000)
			m.Get(key)
		}(i)
	}

	wg.Wait()
}

func BenchmarkShardedMapTS_AsyncGet(b *testing.B) {
	m := NewShardedMapTS[string, int]()

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i)
		m.Set(key, i)
	}

	b.ResetTimer()

	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i%10000)
			m.Get(key)
		}(i)
	}

	wg.Wait()
}

func BenchmarkMapTS_Combined(b *testing.B) {
	m := NewMapTS[string, int]()
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			m.Set(key, i)
		}(i)

		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i%1000)
			m.Get(key)
		}(i)
	}
	wg.Wait()
}

func BenchmarkShardedMapTS_Combined(b *testing.B) {
	m := NewShardedMapTS[string, int]()
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			m.Set(key, i)
		}(i)

		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i%1000)
			m.Get(key)
		}(i)
	}
	wg.Wait()
}
