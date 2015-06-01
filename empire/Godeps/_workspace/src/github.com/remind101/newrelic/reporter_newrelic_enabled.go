// +build newrelic_enabled

package newrelic

import (
	"github.com/remind101/newrelic/sdk"
)

type NRTxReporter struct{}

func (r *NRTxReporter) ReportError(txnID int64, exceptionType, errorMessage, stackTrace, stackFrameDelim string) (int, error) {
	return sdk.TransactionNoticeError(txnID, exceptionType, errorMessage, stackTrace, stackFrameDelim)
}
