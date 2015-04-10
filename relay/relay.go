package relay

var (
	// DefaultOptions is a default Options instance that can be passed when
	// intializing a new Relay.
	DefaultOptions = Options{}
)

type Options struct{}

type Relay struct {
}

// New returns a new Relay instance.
func New(options Options) *Relay {
	return &Relay{}
}
