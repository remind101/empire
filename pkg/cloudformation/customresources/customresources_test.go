package customresources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

var ctx = context.Background()

func TestWithTimeout_NoTimeout(t *testing.T) {
	m := new(mockProvisioner)
	p := WithTimeout(m, time.Second, time.Second)

	m.On("Provision", Request{}).Return("id", nil, nil)

	p.Provision(ctx, Request{})
}

func TestWithTimeout_Timeout_Cleanup(t *testing.T) {
	m := new(mockProvisioner)
	p := WithTimeout(m, time.Millisecond*500, time.Millisecond*500)

	m.On("Provision", Request{}).Return("id", nil, nil).Run(func(mock.Arguments) {
		time.Sleep(time.Millisecond * 750)
	})

	id, _, err := p.Provision(ctx, Request{})
	assert.NoError(t, err)
	assert.Equal(t, "id", id)
}

func TestWithTimeout_GraceTimeout(t *testing.T) {
	m := new(mockProvisioner)
	p := WithTimeout(m, time.Millisecond*500, time.Millisecond*500)

	m.On("Provision", Request{}).Return("id", nil, nil).Run(func(mock.Arguments) {
		time.Sleep(time.Millisecond * 1500)
	})

	_, _, err := p.Provision(ctx, Request{})
	assert.Equal(t, context.DeadlineExceeded, err)
}

type mockProvisioner struct {
	mock.Mock
}

func (m *mockProvisioner) Provision(_ context.Context, req Request) (string, interface{}, error) {
	args := m.Called(req)
	return args.String(0), args.Get(1), args.Error(2)
}

func (m *mockProvisioner) Properties() interface{} {
	return nil
}
