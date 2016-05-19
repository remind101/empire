package kinesumer

import (
	k "github.com/remind101/kinesumer/interface"
)

type Record struct {
	data               []byte
	partitionKey       string
	sequenceNumber     string
	shardId            string
	millisBehindLatest int64
	checkpointC        chan<- k.Record
}

func (r *Record) Data() []byte {
	return r.data
}

func (r *Record) PartitionKey() string {
	return r.partitionKey
}

func (r *Record) SequenceNumber() string {
	return r.sequenceNumber
}

func (r *Record) ShardId() string {
	return r.shardId
}

func (r *Record) MillisBehindLatest() int64 {
	return r.millisBehindLatest
}

func (r *Record) Done() {
	if r.checkpointC != nil {
		r.checkpointC <- r
	}
}
