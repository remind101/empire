package flock

import (
	"os"
	"syscall"
)

// Lock is an implementation of the sync.Locker interface that is backed by a
// file flock(2).
type Lock struct {
	f  *os.File
	fd uintptr
}

// New returns a new Lock instance using the provided file as the backend.
func New(f *os.File) *Lock {
	return &Lock{
		f:  f,
		fd: f.Fd(),
	}
}

// NewPath first creates the file at the given path, then returns a new Lock
// instance backed by the file.
func NewPath(path string) (*Lock, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return New(f), nil
}

// Lock aquires a lock on the file using flock(2). Lock will block if the file
// was previously locked.
func (l *Lock) Lock() {
	if err := syscall.Flock(int(l.fd), syscall.LOCK_EX); err != nil {
		panic(err)
	}
}

// Unlock removes the lock on the file using flock(2).
func (l *Lock) Unlock() {
	if err := syscall.Flock(int(l.fd), syscall.LOCK_UN); err != nil {
		panic(err)
	}
}
