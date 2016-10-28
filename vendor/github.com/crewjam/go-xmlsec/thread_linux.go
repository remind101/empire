package xmlsec

import "syscall"

// getThreadID returns an opaque value that is unique per OS thread
func getThreadID() uintptr {
	return uintptr(syscall.Gettid())
}
