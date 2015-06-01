// +build !newrelic_enabled

// No op implementation for non linux platforms (new relix agent sdk only support linux right now)
package newrelic

import (
	"log"
	"runtime"
)

func Init(app, key string) {
	log.Println("Using NoOp NRTxTracer for unspported platform:", runtime.GOOS, runtime.GOARCH)
	return
}

type NRTxTracer struct{}

func (t *NRTxTracer) BeginTransaction() (int64, error) {
	return 0, nil
}
func (t *NRTxTracer) EndTransaction(txnID int64) error {
	return nil
}
func (t *NRTxTracer) SetTransactionName(txnID int64, name string) error {
	return nil
}
func (t *NRTxTracer) SetTransactionRequestURL(txnID int64, url string) error {
	return nil
}
func (t *NRTxTracer) BeginGenericSegment(txnID, parentID int64, name string) (int64, error) {
	return 0, nil
}
func (t *NRTxTracer) BeginDatastoreSegment(txnID, parentID int64, table, operation, sql, rollupName string) (int64, error) {
	return 0, nil
}
func (t *NRTxTracer) BeginExternalSegment(txnID, parentID int64, host, name string) (int64, error) {
	return 0, nil
}
func (t *NRTxTracer) EndSegment(txnID, parentID int64) error {
	return nil
}
