package newrelic

import (
	"errors"
	"time"
)

var ErrTxAlreadyStarted = errors.New("transaction already started")

// TxTracer handles transaction tracing.
type TxTracer interface {
	BeginTransaction() (int64, error)
	EndTransaction(txnID int64) error

	SetTransactionName(txnID int64, name string) error
	SetTransactionRequestURL(txnID int64, url string) error

	BeginGenericSegment(txnID int64, parentID int64, name string) (int64, error)
	BeginDatastoreSegment(txnID int64, parentID int64, table string, operation string, sql string, rollupName string) (int64, error)
	BeginExternalSegment(txnID int64, parentID int64, host string, name string) (int64, error)
	EndSegment(txnID int64, parentID int64) error
}

// TxReporter reports the first error that occured during a transaction.
type TxReporter interface {
	ReportError(txnID int64, exceptionType, errorMessage, stackTrace, stackFrameDelim string) (int, error)
}

// Recorder handles metrics recording.
type Recorder interface {
	Interval() time.Duration
	Record() error
}

// RecordMetrics records metrics with the default metric recorder.
func RecordMetrics(interval time.Duration) {
	RecordMetricsWithRecorder(newRecorder(interval))
}

// RecordMetricsWithRecorder records metrics with the given recorder.
func RecordMetricsWithRecorder(r Recorder) {
	ticker := time.NewTicker(r.Interval())
	go func() {
		for range ticker.C {
			r.Record()
		}
	}()
}
