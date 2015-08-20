package emptycheckpointer

import (
	k "github.com/remind101/kinesumer/interface"
)

type Checkpointer struct {
}

func (p Checkpointer) DoneC() chan<- k.Record {
	return nil
}

func (p Checkpointer) Begin() error {
	return nil
}

func (p Checkpointer) End() {
}

func (p Checkpointer) GetStartSequence(string) string {
	return ""
}

func (p Checkpointer) Sync() {
}

func (p Checkpointer) TryAcquire(shardID string) error {
	return nil
}

func (p Checkpointer) Release(shardID string) error {
	return nil
}
