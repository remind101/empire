package mocks

import (
	k "github.com/remind101/kinesumer/interface"
	"github.com/stretchr/testify/mock"
)

type Kinesumer struct {
	mock.Mock
}

func (m *Kinesumer) Begin() (int, error) {
	ret := m.Called()

	r0 := ret.Int(0)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesumer) End() {
	m.Called()
}
func (m *Kinesumer) Records() <-chan k.Record {
	ret := m.Called()

	var r0 <-chan k.Record
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(<-chan k.Record)
	}

	return r0
}
