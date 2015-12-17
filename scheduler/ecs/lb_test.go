package ecs

import (
	"testing"

	"github.com/remind101/empire/pkg/lb"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

func TestLBProcessManager_CreateProcess_NoExposure(t *testing.T) {
	p := new(mockProcessManager)
	l := new(mockLBManager)
	m := &LBProcessManager{
		ProcessManager: p,
		lb:             l,
	}

	app := &scheduler.App{}
	process := &scheduler.Process{}

	p.On("CreateProcess", app, process).Return(nil)

	err := m.CreateProcess(context.Background(), app, process)
	assert.NoError(t, err)

	p.AssertExpectations(t)
	l.AssertExpectations(t)
}

func TestLBProcessManager_CreateProcess_NoExistingLoadBalancer(t *testing.T) {
	p := new(mockProcessManager)
	l := new(mockLBManager)
	m := &LBProcessManager{
		ProcessManager: p,
		lb:             l,
	}

	port := int64(8080)
	app := &scheduler.App{
		ID:   "appid",
		Name: "appname",
	}
	process := &scheduler.Process{
		Type:     "web",
		Exposure: scheduler.ExposePrivate,
		Ports: []scheduler.PortMap{
			{Host: &port},
		},
	}

	l.On("LoadBalancers", map[string]string{"AppID": "appid", "ProcessType": "web"}).Return([]*lb.LoadBalancer{}, nil)
	l.On("CreateLoadBalancer", lb.CreateLoadBalancerOpts{
		InstancePort: 8080,
		Tags:         map[string]string{"AppID": "appid", "ProcessType": "web", "App": "appname"},
		External:     false,
	}).Return(&lb.LoadBalancer{
		Name: "lbname",
	}, nil)
	p.On("CreateProcess", app, &scheduler.Process{
		Type:     "web",
		Exposure: scheduler.ExposePrivate,
		Ports: []scheduler.PortMap{
			{Host: &port},
		},
		LoadBalancer: "lbname",
	}).Return(nil)

	err := m.CreateProcess(context.Background(), app, process)
	assert.NoError(t, err)

	p.AssertExpectations(t)
	l.AssertExpectations(t)
}

func TestLBProcessManager_CreateProcess_ExistingLoadBalancer(t *testing.T) {
	p := new(mockProcessManager)
	l := new(mockLBManager)
	m := &LBProcessManager{
		ProcessManager: p,
		lb:             l,
	}

	port := int64(8080)
	app := &scheduler.App{
		ID:   "appid",
		Name: "appname",
	}
	process := &scheduler.Process{
		Type:     "web",
		Exposure: scheduler.ExposePublic,
		Ports: []scheduler.PortMap{
			{Host: &port},
		},
	}

	l.On("LoadBalancers", map[string]string{"AppID": "appid", "ProcessType": "web"}).Return([]*lb.LoadBalancer{
		{Name: "lbname", InstancePort: 8080, External: true},
	}, nil)
	p.On("CreateProcess", app, process).Return(nil)

	err := m.CreateProcess(context.Background(), app, process)
	assert.NoError(t, err)

	p.AssertExpectations(t)
	l.AssertExpectations(t)
}

func TestLBProcessManager_CreateProcess_ExistingLoadBalancer_MismatchedExposure(t *testing.T) {
	p := new(mockProcessManager)
	l := new(mockLBManager)
	m := &LBProcessManager{
		ProcessManager: p,
		lb:             l,
	}

	port := int64(8080)
	app := &scheduler.App{
		ID:   "appid",
		Name: "appname",
	}
	process := &scheduler.Process{
		Type:     "web",
		Exposure: scheduler.ExposePublic,
		Ports: []scheduler.PortMap{
			{Host: &port},
		},
	}

	l.On("LoadBalancers", map[string]string{"AppID": "appid", "ProcessType": "web"}).Return([]*lb.LoadBalancer{
		{Name: "lbname", External: false},
	}, nil)

	err := m.CreateProcess(context.Background(), app, process)
	assert.EqualError(t, err, "Process web is public, but load balancer is private. An update would require me to delete the load balancer.")

	p.AssertExpectations(t)
	l.AssertExpectations(t)
}

func TestLBProcessManager_CreateProcess_ExistingLoadBalancer_NewCert(t *testing.T) {
	p := new(mockProcessManager)
	l := new(mockLBManager)
	m := &LBProcessManager{
		ProcessManager: p,
		lb:             l,
	}

	port := int64(8080)
	app := &scheduler.App{
		ID:   "appid",
		Name: "appname",
	}
	process := &scheduler.Process{
		Type:     "web",
		Exposure: scheduler.ExposePublic,
		Ports: []scheduler.PortMap{
			{Host: &port},
		},
		SSLCert: "newcert",
	}

	l.On("LoadBalancers", map[string]string{"AppID": "appid", "ProcessType": "web"}).Return([]*lb.LoadBalancer{
		{Name: "lbname", External: true, SSLCert: "oldcert"},
	}, nil)

	err := m.CreateProcess(context.Background(), app, process)
	assert.NoError(t, err)

	p.AssertExpectations(t)
	l.AssertExpectations(t)
}

type mockProcessManager struct {
	ProcessManager
	mock.Mock
}

func (m *mockProcessManager) CreateProcess(ctx context.Context, app *scheduler.App, process *scheduler.Process) error {
	args := m.Called(app, process)
	return args.Error(0)
}

type mockLBManager struct {
	mock.Mock
}

func (m *mockLBManager) CreateLoadBalancer(ctx context.Context, opts lb.CreateLoadBalancerOpts) (*lb.LoadBalancer, error) {
	args := m.Called(opts)
	return args.Get(0).(*lb.LoadBalancer), args.Error(1)
}

func (m *mockLBManager) DestroyLoadBalancer(ctx context.Context, lb *lb.LoadBalancer) error {
	args := m.Called(lb)
	return args.Error(0)
}

func (m *mockLBManager) LoadBalancers(ctx context.Context, tags map[string]string) ([]*lb.LoadBalancer, error) {
	args := m.Called(tags)
	return args.Get(0).([]*lb.LoadBalancer), args.Error(1)
}
