package lock

import (
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/remind101/empire/dbtest"
	"github.com/stretchr/testify/assert"
)

const testKey = 1234

func TestAdvisoryLock(t *testing.T) {
	db := dbtest.Open(t)
	defer db.Close()

	l, err := NewAdvisoryLock(db, testKey)
	assert.NoError(t, err)

	err = l.Lock()
	assert.NoError(t, err)

	err = l.Unlock()
	assert.NoError(t, err)
}

func TestAdvisoryLock_Locked(t *testing.T) {
	db := dbtest.Open(t)
	defer db.Close()

	a, err := NewAdvisoryLock(db, testKey)
	assert.NoError(t, err)

	b, err := NewAdvisoryLock(db, testKey)
	assert.NoError(t, err)

	bLocked := make(chan error)

	err = a.Lock()
	assert.NoError(t, err)
	t.Log("A locked")

	go func() {
		bLocked <- b.Lock()
		t.Log("B locked")
	}()

	select {
	case <-bLocked:
		t.Fatal("b should not be locked at this time")
	case <-time.After(time.Second):
	}

	err = a.Unlock()
	assert.NoError(t, err)
	t.Log("A unlocked")

	select {
	case err := <-bLocked:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for b to obtain the lock")
	}

	err = b.Unlock()
	assert.NoError(t, err)
	t.Log("B unlocked")
}

func TestAdvisoryLock_CancelPending(t *testing.T) {
	db := dbtest.Open(t)
	defer db.Close()

	a, err := NewAdvisoryLock(db, testKey)
	a.Context = "A"
	assert.NoError(t, err)

	b, err := NewAdvisoryLock(db, testKey)
	b.Context = "B"
	assert.NoError(t, err)

	c, err := NewAdvisoryLock(db, testKey)
	c.Context = "C"
	assert.NoError(t, err)

	bLocked := make(chan error)
	cLocked := make(chan error)

	err = a.Lock()
	assert.NoError(t, err)
	t.Log("A locked")

	go func() {
		bLocked <- b.Lock()
		t.Log("B locked")
	}()

	go func() {
		time.Sleep(time.Second)
		t.Log("Canceling pending locks")
		err = c.CancelPending()
		err := a.Unlock()
		assert.NoError(t, err)
		cLocked <- c.Lock()
		t.Log("C locked")
	}()

	select {
	case err := <-bLocked:
		assert.Equal(t, Canceled, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for b to obtain the lock")
	}

	select {
	case err := <-cLocked:
		assert.NoError(t, err)
		c.Unlock()
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for c to obtain the lock")
	}
}

func TestAdvisoryLock_Timeout(t *testing.T) {
	db := dbtest.Open(t)
	defer db.Close()

	a, err := NewAdvisoryLock(db, testKey)
	assert.NoError(t, err)
	b, err := NewAdvisoryLock(db, testKey)
	b.LockTimeout = time.Second
	assert.NoError(t, err)

	err = a.Lock()
	assert.NoError(t, err)

	err = b.Lock()
	assert.Equal(t, ErrLockTimeout, err)

	err = a.Unlock()
	assert.NoError(t, err)
}

func TestAdvisoryLock_Unlocked(t *testing.T) {
	db := dbtest.Open(t)
	defer db.Close()

	l, err := NewAdvisoryLock(db, testKey)
	assert.NoError(t, err)

	err = l.Lock()
	assert.NoError(t, err)
	err = l.Lock()
	assert.NoError(t, err)

	err = l.Unlock()
	assert.NoError(t, err)
	err = l.Unlock()
	assert.NoError(t, err)

	// Unlocking an already unlocked advisory lock should panic.
	defer func() {
		v := recover()
		assert.NotNil(t, v)
	}()
	err = l.Unlock()
	assert.NoError(t, err)
}

func TestAdvisoryLock_Used(t *testing.T) {
	db := dbtest.Open(t)
	defer db.Close()

	l, err := NewAdvisoryLock(db, testKey)
	assert.NoError(t, err)

	err = l.Lock()
	assert.NoError(t, err)
	err = l.Unlock()
	assert.NoError(t, err)

	// Locking a used lock should panic
	defer func() {
		v := recover()
		assert.NotNil(t, v)
	}()
	err = l.Lock()
	assert.NoError(t, err)
}
