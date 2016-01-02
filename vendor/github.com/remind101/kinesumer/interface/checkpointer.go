package kinesumeriface

type Checkpointer interface {
	DoneC() chan<- Record
	Begin() error
	End()
	GetStartSequence(shardID string) string
	Sync()
}
