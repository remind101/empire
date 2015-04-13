package relay

import (
	"sync"

	"code.google.com/p/go-uuid/uuid"
)

var (
	// DefaultOptions is a default Options instance that can be passed when
	// intializing a new Relay.
	DefaultOptions = Options{}
)

type Options struct{}

type Relay struct {
	sync.Mutex
	sessions map[string]bool
}

// New returns a new Relay instance.
func New(options Options) *Relay {
	return &Relay{
		sessions: map[string]bool{},
	}
}

func (r *Relay) NewSession() string {
	r.Lock()
	defer r.Unlock()
	id := uuid.New()
	r.sessions[id] = true
	return id
}
