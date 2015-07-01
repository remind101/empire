# Go flock(2) sync.Locker

This is a simple implementation of the `sync.Locker` interface backed by the
**[flock(2)](http://linux.die.net/man/2/flock)** syscall.

I wouldn't recommend using this in production, but it can be handy if you need a
quick inter process lock to prevent concurrent access to shared resources in
tests.

## Usage

```go
l := flock.NewPath("/tmp/mylock.lock")
l.Lock()
defer l.Unlock()
```
