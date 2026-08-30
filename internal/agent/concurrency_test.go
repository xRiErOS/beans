package agent

import (
	"fmt"
	"sync"
	"testing"
)

// TestManagerConcurrentSessionAccess hammers one session from several
// goroutines at once. It covers the paths a live server takes: a subscription
// reading state while messages arrive and the mode is toggled, and a clear
// landing in the middle of it. Plain `just test` only trips Go's built-in
// concurrent-map detector here — run it under `just test-race` for the
// slice-append and struct-field races.
func TestManagerConcurrentSessionAccess(t *testing.T) {
	m := NewManager("", nil)
	defer m.Shutdown()

	const beanID = "bean-concurrent"
	const rounds = 50

	var wg sync.WaitGroup
	start := make(chan struct{})

	worker := func(fn func(i int)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := range rounds {
				fn(i)
			}
		}()
	}

	worker(func(i int) { m.AddInfoMessage(beanID, fmt.Sprintf("info-%d", i)) })
	worker(func(i int) { _ = m.SetPlanMode(beanID, i%2 == 0) })
	worker(func(i int) { _ = m.SetEffort(beanID, "medium") })
	worker(func(int) { _ = m.GetSession(beanID) })
	worker(func(int) { _ = m.ListRunningSessions() })
	worker(func(int) {
		ch := m.Subscribe(beanID)
		m.Unsubscribe(beanID, ch)
	})
	worker(func(i int) {
		if i%10 == 0 {
			_ = m.ClearSession(beanID)
		}
	})

	close(start)
	wg.Wait()

	// The session must still be coherent: readable, with no message left
	// half-written by a concurrent append.
	if s := m.GetSession(beanID); s != nil {
		for _, msg := range s.Messages {
			if msg.Role == "" {
				t.Fatalf("session holds a message with no role: %+v", msg)
			}
		}
	}
}

// TestManagerConcurrentSubscribeNotify covers subscribers coming and going
// while notifications are being fanned out.
func TestManagerConcurrentSubscribeNotify(t *testing.T) {
	m := NewManager("", nil)
	defer m.Shutdown()

	const beanID = "bean-subs"
	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 25 {
				ch := m.Subscribe(beanID)
				global := m.SubscribeGlobal()
				m.AddInfoMessage(beanID, "ping")
				m.Unsubscribe(beanID, ch)
				m.UnsubscribeGlobal(global)
			}
		}()
	}

	wg.Wait()
}
