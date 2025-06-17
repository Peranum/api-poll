package channel_test

import (
	"errors"
	"testing"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/channel"
)

func TestBroadcastFollowAndUnfollow(t *testing.T) {
	b := channel.NewBroadcast[int](0)

	ch, err := b.Follow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = b.Unfollow(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if b.Len() != 0 {
		t.Fatalf("expected 0 followers, got %d", b.Len())
	}
}

func TestBroadcastSend(t *testing.T) {
	b := channel.NewBroadcast[int](0)

	ch, err := b.Follow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer b.Unfollow(ch)

	ch2, err := b.Follow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer b.Unfollow(ch2)

	err = b.Send(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value := <-ch
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}

	value = <-ch2
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}

	ch3, err := b.Follow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer b.Unfollow(ch3)
}

func TestBroadcastClose(t *testing.T) {
	b := channel.NewBroadcast[int](0)

	ch, err := b.Follow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer b.Unfollow(ch)

	b.Close()
	if !b.Closed() {
		t.Fatal("expected broadcast to be closed")
	}

	err = b.Send(42)
	if err == nil || !errors.Is(err, channel.ErrBroadcastClosed) {
		t.Fatalf("expected ErrBroadcastClosed, got %v", err)
	}
}

func TestBroadcastLenAndCap(t *testing.T) {
	b := channel.NewBroadcast[int](10)

	if b.Len() != 0 {
		t.Fatalf("expected 0 followers, got %d", b.Len())
	}
	if b.Cap() != 10 {
		t.Fatalf("expected capacity 10, got %d", b.Cap())
	}

	ch, err := b.Follow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer b.Unfollow(ch)

	if b.Len() != 1 {
		t.Fatalf("expected 1 follower, got %d", b.Len())
	}

	ch, err = b.Follow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer b.Unfollow(ch)

	if b.Len() != 2 {
		t.Fatalf("expected 2 followers, got %d", b.Len())
	}
}
