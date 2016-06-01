package lock

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"
)

// ErrLockTimeout is returned when obtaining the lock times out.
var ErrLockTimeout = errors.New("timed out waiting for lock")

// Canceled is returned when another process cancels this lock.
var Canceled = errors.New("lock canceled by another process")

// AdvisoryLock wraps PostgresSQL advisory locks to act like a sync.Locker.
type AdvisoryLock struct {
	// This will be added as a comment string to the query to obtain the
	// lock, which can be useful in debugging.
	Context string

	// An optional timeout for obtaining the lock. The zero value is no
	// timeout.
	LockTimeout time.Duration

	// The advisory lock key.
	key uint32

	// Protects individual calls to Lock and Unlock.
	mu sync.Mutex
	// This is the transaction that the lock is held within. We need to use
	// a transaction to ensure that we hold a single connection when
	// performing queries.
	tx       *sql.Tx
	c        int
	commited bool
}

// NewAdvisoryLock opens a new transaction and returns the AdvisoryLock.
func NewAdvisoryLock(db *sql.DB, key uint32) (*AdvisoryLock, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	return &AdvisoryLock{
		tx:  tx,
		key: key,
	}, nil
}

// CancelPending cancels any pending advisory locks (those that are not yet
// granted) for this advisory lock key.
//
// This can be useful in situations where you only want a maximum of 1 process
// waiting to obtain the lock at any time.
func (l *AdvisoryLock) CancelPending() error {
	_, err := l.tx.Exec(`SELECT pg_cancel_backend(pending.pid)
				FROM (SELECT pid 
					FROM pg_locks 
					WHERE locktype = 'advisory' 
					AND granted = 'f' 
					AND objid = $1) as pending`, l.key)
	if err != nil {
		return fmt.Errorf("error canceling pending locks: %v", err)
	}
	return nil
}

// Lock obtains the advisory lock.
func (l *AdvisoryLock) Lock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.commited {
		panic("lock called on commited lock")
	}

	if l.LockTimeout != 0 {
		_, err := l.tx.Exec(fmt.Sprintf("SET LOCAL lock_timeout = %d", int(l.LockTimeout.Seconds()*1000)))
		if err != nil {
			return fmt.Errorf("error setting lock timeout: %v", err)
		}
	}

	_, err := l.tx.Exec(fmt.Sprintf("SELECT pg_advisory_lock($1) /* %s */", l.Context), l.key)
	if err != nil {
		// If there's an error trying to obtain the lock, probably the
		// safest thing to do is commit the transaction and make this
		// lock invalid.
		l.commit()

		// This will happen when a newer stack update obsoletes
		// this one. We simply return nil.
		if err, ok := err.(*pq.Error); ok {
			switch err.Code.Name() {
			case "query_canceled":
				return Canceled
			case "lock_not_available":
				return ErrLockTimeout
			}
		}

		return fmt.Errorf("error obtaining lock: %v", err)
	}

	l.c += 1

	return nil
}

// Unlock releases the advisory lock.
func (l *AdvisoryLock) Unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.commited {
		panic("unlock called on commited lock")
	}

	if l.c == 0 {
		panic("unlock of unlocked advisory lock")
	}

	_, err := l.tx.Exec(fmt.Sprintf("SELECT pg_advisory_unlock($1) /* %s */", l.Context), l.key)
	if err != nil {
		return err
	}

	l.c -= 1

	if l.c == 0 {
		if err := l.commit(); err != nil {
			return err
		}
	}

	return err
}

func (l *AdvisoryLock) commit() error {
	l.commited = true
	return l.tx.Commit()
}

// Locker returns a sync.Locker compatible version of this AdvisoryLock.
func (l *AdvisoryLock) Locker() sync.Locker {
	return &locker{l}
}

// locker wraps an AdvisoryLock to implement the sync.Locker interface.
type locker struct {
	*AdvisoryLock
}

func (l *locker) Lock() {
	if err := l.AdvisoryLock.Lock(); err != nil {
		panic(err)
	}
}

func (l *locker) Unlock() {
	if err := l.AdvisoryLock.Unlock(); err != nil {
		panic(err)
	}
}
