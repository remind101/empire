package mocks

import (
	"time"

	"github.com/stretchr/testify/mock"
)

type Provisioner struct {
	mock.Mock
}

func (m *Provisioner) TryAcquire(shardID string) error {
	ret := m.Called(shardID)

	r0 := ret.Error(0)

	return r0
}
func (m *Provisioner) Release(shardID string) error {
	ret := m.Called(shardID)

	r0 := ret.Error(0)

	return r0
}
func (m *Provisioner) Heartbeat(shardID string) error {
	ret := m.Called(shardID)

	r0 := ret.Error(0)

	return r0
}
func (m *Provisioner) TTL() time.Duration {
	ret := m.Called()

	r0 := ret.Get(0).(time.Duration)

	return r0
}
