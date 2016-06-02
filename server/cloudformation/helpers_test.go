package cloudformation

import (
	"golang.org/x/net/context"

	"github.com/remind101/empire"
	"github.com/stretchr/testify/mock"
)

type mockEmpire struct {
	empire.Empire
	mock.Mock
}

func (m *mockEmpire) Create(_ context.Context, opts empire.CreateOpts) (*empire.App, error) {
	args := m.Called(opts)
	return args.Get(0).(*empire.App), args.Error(1)
}

func (m *mockEmpire) AppsFind(q empire.AppsQuery) (*empire.App, error) {
	args := m.Called(q)
	return args.Get(0).(*empire.App), args.Error(1)
}

func (m *mockEmpire) Destroy(_ context.Context, opts empire.DestroyOpts) error {
	args := m.Called(opts)
	return args.Error(0)
}

func (m *mockEmpire) Set(_ context.Context, opts empire.SetOpts) (*empire.Config, error) {
	args := m.Called(opts)
	return args.Get(0).(*empire.Config), args.Error(1)
}
