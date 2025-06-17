package container

import (
	"bytes"
	"encoding/gob"
	"hash/maphash"
	"unsafe"
)

const (
	ShardedMapDefaultShardCapacity = 512
	ShardedMapTSDefaultShardsCount = MapTSDefaultCapacity
)

// Fixed seed for deterministic hashing
var hashSeed = maphash.MakeSeed()

type ShardedMapTS[K comparable, V any] struct {
	shards []*MapTS[K, V]
}

func NewShardedMapTS[K comparable, V any](
	sizes ...int,
) *ShardedMapTS[K, V] {
	shardCapacity, shardsCount := parseShardedMapTSSizes(sizes)

	shards := make([]*MapTS[K, V], shardsCount)
	for i := 0; i < shardsCount; i++ {
		shards[i] = NewMapTS[K, V](shardCapacity)
	}

	return &ShardedMapTS[K, V]{
		shards: shards,
	}
}

func (c *ShardedMapTS[K, V]) Set(key K, value V) {
	shard := c.getShard(key)
	shard.Set(key, value)
}

func (c *ShardedMapTS[K, V]) GetOK(key K) (V, bool) {
	shard := c.getShard(key)

	return shard.GetOK(key)
}

func (c *ShardedMapTS[K, V]) Get(key K) V {
	shard := c.getShard(key)

	return shard.Get(key)
}

func (c *ShardedMapTS[K, V]) Has(key K) bool {
	shard := c.getShard(key)

	return shard.Has(key)
}

func (c *ShardedMapTS[K, V]) Delete(key K) {
	shard := c.getShard(key)
	shard.Delete(key)
}

func (c *ShardedMapTS[K, V]) Clear() {
	for _, shard := range c.shards {
		shard.Clear()
	}
}

func (c *ShardedMapTS[K, V]) Size() int {
	totalSize := 0
	for _, shard := range c.shards {
		totalSize += shard.Size()
	}
	return totalSize
}

func (c *ShardedMapTS[K, V]) ShardsUnsafe() []map[K]V {
	shards := make([]map[K]V, len(c.shards))

	for i, shard := range c.shards {
		shards[i] = shard.MapUnsafe()
	}

	return shards
}

func (c *ShardedMapTS[K, V]) Lock() {
	for _, shard := range c.shards {
		shard.Lock()
	}
}

func (c *ShardedMapTS[K, V]) Unlock() {
	for _, shard := range c.shards {
		shard.Unlock()
	}
}

func (c *ShardedMapTS[K, V]) RLock() {
	for _, shard := range c.shards {
		shard.RLock()
	}
}

func (c *ShardedMapTS[K, V]) RUnlock() {
	for _, shard := range c.shards {
		shard.RUnlock()
	}
}

func (c *ShardedMapTS[K, V]) ForEach(fn func(shard map[K]V, key K, value V)) {
	c.Lock()
	for _, shard := range c.ShardsUnsafe() {
		for key, value := range shard {
			fn(shard, key, value)
		}
	}
	c.Unlock()
}

func (c *ShardedMapTS[K, V]) getShard(key K) *MapTS[K, V] {
	index := c.getHash(key) % uint64(len(c.shards))

	return c.shards[index]
}

func (c *ShardedMapTS[K, V]) getHash(key K) uint64 {
	switch k := any(key).(type) {
	case string:
		return maphash.String(hashSeed, k)
	case int:
		b := (*[8]byte)(unsafe.Pointer(&k))[:8]
		return maphash.Bytes(hashSeed, b)
	case int64:
		b := (*[8]byte)(unsafe.Pointer(&k))[:8]
		return maphash.Bytes(hashSeed, b)
	case int32:
		b := (*[4]byte)(unsafe.Pointer(&k))[:4]
		return maphash.Bytes(hashSeed, b)
	case int16:
		b := (*[2]byte)(unsafe.Pointer(&k))[:2]
		return maphash.Bytes(hashSeed, b)
	case int8:
		b := (*[1]byte)(unsafe.Pointer(&k))[:1]
		return maphash.Bytes(hashSeed, b)
	case uint:
		b := (*[8]byte)(unsafe.Pointer(&k))[:8]
		return maphash.Bytes(hashSeed, b)
	case uint64:
		b := (*[8]byte)(unsafe.Pointer(&k))[:8]
		return maphash.Bytes(hashSeed, b)
	case uint32:
		b := (*[4]byte)(unsafe.Pointer(&k))[:4]
		return maphash.Bytes(hashSeed, b)
	case uint16:
		b := (*[2]byte)(unsafe.Pointer(&k))[:2]
		return maphash.Bytes(hashSeed, b)
	case uint8:
		b := (*[1]byte)(unsafe.Pointer(&k))[:1]
		return maphash.Bytes(hashSeed, b)
	case float64:
		b := (*[8]byte)(unsafe.Pointer(&k))[:8]
		return maphash.Bytes(hashSeed, b)
	case float32:
		b := (*[4]byte)(unsafe.Pointer(&k))[:4]
		return maphash.Bytes(hashSeed, b)
	case bool:
		var b [1]byte
		if k {
			b[0] = 1
		}
		return maphash.Bytes(hashSeed, b[:])
	default:
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		enc.Encode(key)
		return maphash.Bytes(hashSeed, buf.Bytes())
	}
}

func parseShardedMapTSSizes(sizes []int) (int, int) {
	shardCapacity := ShardedMapDefaultShardCapacity
	shardsCount := ShardedMapTSDefaultShardsCount

	if len(sizes) > 0 {
		shardCapacity = sizes[0]
	}

	if len(sizes) > 1 {
		shardsCount = sizes[1]
	}

	if shardCapacity <= 0 {
		shardCapacity = ShardedMapDefaultShardCapacity
	}

	if shardsCount <= 0 {
		shardsCount = ShardedMapTSDefaultShardsCount
	}

	return shardCapacity, shardsCount
}
