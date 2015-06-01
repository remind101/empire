package newrelic

import "golang.org/x/net/context"

// Tx represents a transaction.
type Tx interface {
	Start() error
	End() error
	StartGeneric(name string) error
	StartDatastore(table, operation, sql, rollupName string) error
	StartExternal(host, name string) error
	EndSegment() error
	ReportError(exceptionType, errorMessage, stackTrace, stackFrameDelim string) error
}

// tx implements the Tx interface.
type tx struct {
	Tracer   TxTracer
	Reporter TxReporter

	id   int64
	name string
	url  string
	ss   *SegmentStack
}

// NewTx returns a new transaction.
func NewTx(name string) *tx {
	return &tx{
		Tracer:   &NRTxTracer{},
		Reporter: &NRTxReporter{},
		name:     name,
		ss:       NewSegmentStack(),
	}
}

// NewRequestTx returns a new transaction with a request url.
func NewRequestTx(name string, url string) *tx {
	t := NewTx(name)
	t.url = url
	return t
}

// Start starts a transaction, setting the id.
func (t *tx) Start() error {
	var err error

	if t.id != 0 {
		return ErrTxAlreadyStarted
	}
	if t.id, err = t.Tracer.BeginTransaction(); err != nil {
		return err
	}
	if err = t.Tracer.SetTransactionName(t.id, t.name); err != nil {
		return err
	}
	if t.url != "" {
		return t.Tracer.SetTransactionRequestURL(t.id, t.url)
	}
	return nil
}

// End ends a transaction.
func (t *tx) End() error {
	for t.ss.Peek() != rootSegment {
		t.EndSegment()
	}
	return t.Tracer.EndTransaction(t.id)
}

// StartGeneric starts a generic segment.
func (t *tx) StartGeneric(name string) error {
	id, err := t.Tracer.BeginGenericSegment(t.id, t.ss.Peek(), name)
	if err != nil {
		return err
	}
	t.ss.Push(id)
	return nil
}

// StartDatastore starts a datastore segment.
func (t *tx) StartDatastore(table, operation, sql, rollupName string) error {
	id, err := t.Tracer.BeginDatastoreSegment(t.id, t.ss.Peek(), table, operation, sql, rollupName)
	if err != nil {
		return err
	}
	t.ss.Push(id)
	return nil
}

// StartExternal starts an external segment.
func (t *tx) StartExternal(host, name string) error {
	id, err := t.Tracer.BeginExternalSegment(t.id, t.ss.Peek(), host, name)
	if err != nil {
		return err
	}
	t.ss.Push(id)
	return nil
}

// EndSegment ends the segment at the top of the stack.
func (t *tx) EndSegment() error {
	if id, ok := t.ss.Pop(); ok {
		return t.Tracer.EndSegment(t.id, id)
	}
	return nil
}

// ReportError reports an error that occured during the transaction.
func (t *tx) ReportError(exceptionType, errorMessage, stackTrace, stackFrameDelim string) error {
	_, err := t.Reporter.ReportError(t.id, exceptionType, errorMessage, stackTrace, stackFrameDelim)
	return err
}

// WithTx inserts a newrelic.Tx into the provided context.
func WithTx(ctx context.Context, t Tx) context.Context {
	return context.WithValue(ctx, txKey, t)
}

// FromContext returns a newrelic.Tx from the context.
func FromContext(ctx context.Context) (Tx, bool) {
	t, ok := ctx.Value(txKey).(Tx)
	return t, ok
}

type key int

const (
	txKey key = iota
)
