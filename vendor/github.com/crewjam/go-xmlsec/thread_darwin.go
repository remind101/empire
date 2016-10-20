package xmlsec

import "unsafe"

// #include <pthread.h>
import "C"

// getThreadID returns an opaque value that is unique per OS thread.
func getThreadID() uintptr {
	// Darwin lacks a meaningful version of gettid() so instead we use
	// ptread_self() as a proxy.
	return uintptr(unsafe.Pointer(C.pthread_self()))
}
