package beancore

import (
	"fmt"
	"testing"
)

// TestFanOutResyncAfterOverflow covers what a subscriber is told after its
// channel overflowed. Dropping the events silently left the client's state
// permanently behind the store; it now receives a resync marker, which the
// GraphQL layer answers with a fresh full snapshot.
func TestFanOutResyncAfterOverflow(t *testing.T) {
	core, _ := setupTestCore(t)

	ch, unsub := core.Subscribe()
	defer unsub()

	// Overflow the subscriber's buffer without reading.
	for i := range 40 {
		core.fanOut([]BeanEvent{{Type: EventUpdated, BeanID: fmt.Sprintf("b%d", i)}})
	}

	// Drain the buffer. The marker has to be waiting in there without any
	// further fan-out: a client that fell behind on the last burst of a busy
	// period would otherwise never be told to resync.
	var last []BeanEvent
	drained := 0
	for len(ch) > 0 {
		last = <-ch
		drained++
	}
	if drained == 0 {
		t.Fatal("expected buffered events to drain")
	}
	if len(last) != 1 || last[0].Type != EventResync {
		t.Fatalf("last buffered batch = %+v, want a single resync event", last)
	}

	// Normal service resumes afterwards, with no second marker.
	core.fanOut([]BeanEvent{{Type: EventUpdated, BeanID: "after-overflow"}})

	batch := <-ch
	if len(batch) != 1 || batch[0].BeanID != "after-overflow" {
		t.Fatalf("batch after the resync = %+v, want the update for after-overflow", batch)
	}
}

// TestFanOutNoResyncWithoutOverflow keeps the marker out of the normal path.
func TestFanOutNoResyncWithoutOverflow(t *testing.T) {
	core, _ := setupTestCore(t)

	ch, unsub := core.Subscribe()
	defer unsub()

	core.fanOut([]BeanEvent{{Type: EventUpdated, BeanID: "b1"}})

	batch := <-ch
	if len(batch) != 1 || batch[0].BeanID != "b1" {
		t.Fatalf("batch = %+v, want the update for b1", batch)
	}
	if len(ch) != 0 {
		t.Errorf("channel holds %d further batches, want none", len(ch))
	}
}

// TestEventResyncString keeps the marker readable in logs.
func TestEventResyncString(t *testing.T) {
	if got, want := EventResync.String(), "resync"; got != want {
		t.Errorf("EventResync.String() = %q, want %q", got, want)
	}
}
