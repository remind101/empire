package flock

import (
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	l1 := newTestLock(t)
	l1.Lock()

	aquired := make(chan struct{}, 1)
	l2 := newTestLock(t)
	go func() {
		l2.Lock()
		aquired <- struct{}{}
	}()

	select {
	case <-aquired:
		t.Fatal("Lock should not have been aquired")
	case <-time.After(100 * time.Millisecond):
		// Timed out, which means the lock is probably working.
	}

	aquired = make(chan struct{}, 1)
	go func() {
		l2.Lock()
		aquired <- struct{}{}
	}()

	l1.Unlock()

	select {
	case <-aquired:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out while aquiring lock")
	}
}

var testFile = "/tmp/flock.test.lock"

func newTestLock(t testing.TB) *Lock {
	l, err := NewPath(testFile)
	if err != nil {
		t.Fatal(err)
	}

	return l
}
