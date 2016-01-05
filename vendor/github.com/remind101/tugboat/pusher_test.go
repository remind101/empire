package tugboat

import (
	"testing"
	"time"
)

type fakePusher struct {
	events chan string
}

func (p *fakePusher) Publish(data, event string, channels ...string) error {
	p.events <- event
	return nil
}

func TestAsyncPusher(t *testing.T) {
	f := &fakePusher{events: make(chan string, 1)}
	p := newAsyncPusher(f, 1)

	if err := p.Publish("data", "event", "channel"); err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-f.events:
		if got, want := e, "event"; got != want {
			t.Fatalf("Event => %s; want %s", got, want)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout")
	}
}
