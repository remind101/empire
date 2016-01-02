package mocks

import (
	k "github.com/remind101/kinesumer/interface"
	"github.com/stretchr/testify/mock"
)

type Checkpointer struct {
	mock.Mock
}

func (m *Checkpointer) DoneC() chan<- k.Record {
	ret := m.Called()

	var r0 chan k.Record
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(chan k.Record)
	}

	return r0
}
func (m *Checkpointer) Begin() error {
	ret := m.Called()

	r0 := ret.Error(0)

	return r0
}
func (m *Checkpointer) End() {
	m.Called()
}
func (m *Checkpointer) GetStartSequence(shardID string) string {
	ret := m.Called(shardID)

	r0 := ret.String(0)

	return r0
}
func (m *Checkpointer) Sync() {
	m.Called()
}
func (m *Checkpointer) TryAcquire(shardID string) error {
	ret := m.Called(shardID)

	r0 := ret.Error(0)

	return r0
}
func (m *Checkpointer) Release(shardID string) error {
	ret := m.Called(shardID)

	r0 := ret.Error(0)

	return r0
}
