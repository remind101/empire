package kinesumeriface

type Record interface {
	Data() []byte
	PartitionKey() string
	SequenceNumber() string
	ShardId() string
	MillisBehindLatest() int64
	Done()
}
