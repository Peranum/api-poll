package channel

import (
	"errors"
	"sync"
	"sync/atomic"
)

const (
	BroadcastFollowerDefaultCapacity = 16
	BroadcastDefaultCapacity         = 16
)

var (
	ErrBroadcastClosed  = errors.New("broadcast closed")
	ErrFollowerNotFound = errors.New("follower not found")
)

type Broadcast[T any] struct {
	followers      []chan T
	followersGuard sync.Mutex

	capacity int
	closed   atomic.Bool
}

func NewBroadcast[T any](capacity int) *Broadcast[T] {
	if capacity <= 0 {
		capacity = BroadcastDefaultCapacity
	}

	return &Broadcast[T]{
		followers: make([]chan T, 0, BroadcastFollowerDefaultCapacity),
		capacity:  capacity,
	}
}

func (b *Broadcast[T]) Follow() (<-chan T, error) {
	return b.follow(b.capacity)
}

func (b *Broadcast[T]) FollowWithMemory(memory []T) (<-chan T, error) {
	ch, err := b.follow(max(b.capacity, len(memory)+len(memory)>>3+1))
	if err != nil {
		return nil, err
	}

	for _, value := range memory {
		ch <- value
	}

	return ch, nil
}

func (b *Broadcast[T]) follow(minRequiredCapacity int) (chan T, error) {
	b.followersGuard.Lock()
	defer b.followersGuard.Unlock()

	if b.closed.Load() {
		return nil, ErrBroadcastClosed
	}

	ch := make(chan T, minRequiredCapacity)

	b.followers = append(b.followers, ch)

	return ch, nil
}

func (b *Broadcast[T]) Unfollow(ch <-chan T) error {
	b.followersGuard.Lock()
	defer b.followersGuard.Unlock()

	for i, follower := range b.followers {
		if follower == ch {
			close(follower)
			b.followers[i] = b.followers[len(b.followers)-1]
			b.followers = b.followers[:len(b.followers)-1]

			return nil
		}
	}

	return ErrFollowerNotFound
}

func (b *Broadcast[T]) Send(value T) error {
	b.followersGuard.Lock()
	defer b.followersGuard.Unlock()

	if b.closed.Load() {
		return ErrBroadcastClosed
	}

	for _, follower := range b.followers {
		follower <- value
	}

	return nil
}

func (b *Broadcast[T]) Close() {
	b.followersGuard.Lock()
	defer b.followersGuard.Unlock()

	if b.closed.CompareAndSwap(false, true) {
		for _, follower := range b.followers {
			close(follower)
		}

		b.followers = nil
	}
}

func (b *Broadcast[T]) Closed() bool {
	return b.closed.Load()
}

func (b *Broadcast[T]) Len() int {
	b.followersGuard.Lock()
	defer b.followersGuard.Unlock()

	return len(b.followers)
}

func (b *Broadcast[T]) Cap() int {
	return b.capacity
}

func NewBroadcastAdapter[T any](ch <-chan T) *Broadcast[T] {
	b := NewBroadcast[T](cap(ch))

	go func() {
		defer b.Close()

		for v := range ch {
			b.Send(v)
		}
	}()

	return b
}
