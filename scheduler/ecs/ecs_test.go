package ecs

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/remind101/empire/pkg/awsutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/scheduler/ecs/lb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

func TestScheduler_Submit(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":"empire"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":["arn:aws:ecs:us-east-1:249285743859:service/1234--web"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
				Body:       `{"cluster":"empire","services":["arn:aws:ecs:us-east-1:249285743859:service/1234--web"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"services":[{"taskDefinition":"1234--web"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"cpu":128,"command":["acme-inc", "web", "--port 80"],"environment":[{"name":"USER","value":"foo"},{"name":"PORT","value":"8080"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web"}]}}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
				Body:       `{"containerDefinitions":[{"cpu":128,"command":["acme-inc", "web", "--port", "80"],"environment":[{"name":"USER","value":"foo"},{"name":"PORT","value":"8080"}],"dockerLabels":{"label1":"foo","label2":"bar"},"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web","portMappings":[{"containerPort":8080,"hostPort":8080}]}],"family":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       "",
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"empire","desiredCount":0,"service":"1234--web","taskDefinition":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"service": {}}`,
			},
		},
	})
	m, s := newTestScheduler(h)
	defer s.Close()

	m.lb.(*mockLBManager).On("LoadBalancers", map[string]string{
		"AppID":       "1234",
		"ProcessType": "web",
	}).Return([]*lb.LoadBalancer{
		{Name: "lb-1234", InstancePort: 8080},
	}, nil)

	if err := m.Submit(context.Background(), fakeApp); err != nil {
		t.Fatal(err)
	}
}

func TestScheduler_Scale(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"empire","desiredCount":10,"service":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
	})
	m, s := newTestScheduler(h)
	defer s.Close()

	if err := m.Scale(context.Background(), "1234", "web", 10); err != nil {
		t.Fatal(err)
	}
}

func TestScheduler_Instances(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":"empire"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":["arn:aws:ecs:us-east-1:249285743859:service/1234--web"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
				Body:       `{"cluster":"empire","serviceName":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskArns":["arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570","arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74571"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
				Body:       `{"cluster":"empire","startedBy":"1234"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskArns":[]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
				Body:       `{"cluster":"empire","tasks":["arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570","arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74571"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"tasks":[{"taskArn":"arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570","taskDefinitionArn":"arn:aws:ecs:us-east-1:249285743859:task-definition/1234--web","lastStatus":"RUNNING","startedAt": 1448419193},{"taskArn":"arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74571","taskDefinitionArn":"arn:aws:ecs:us-east-1:249285743859:task-definition/1234--web","lastStatus":"RUNNING","startedAt": 1448419193}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"arn:aws:ecs:us-east-1:249285743859:task-definition/1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"name":"web","cpu":256,"memory":256,"command":["acme-inc", "web", "--port", "80"]}]}}`,
			},
		},
	})
	m, s := newTestScheduler(h)
	defer s.Close()

	instances, err := m.Instances(context.Background(), "1234")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 2 {
		t.Fatal("expected 2 instances")
	}

	i := instances[0]

	if got, want := i.State, "RUNNING"; got != want {
		t.Fatalf("State => %s; want %s", got, want)
	}

	if got, want := i.UpdatedAt, time.Unix(1448419193, 0).UTC(); got != want {
		t.Fatalf("UpdatedAt => %s; want %s", got, want)
	}

	if got, want := i.Process.Command, []string{"acme-inc", "web", "--port", "80"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Command => %s; want %s", got, want)
	}

	if got, want := i.Process.Type, "web"; got != want {
		t.Fatalf("Type => %s; want %s", got, want)
	}
}

func TestScheduler_Remove(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":"empire"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":["arn:aws:ecs:us-east-1:249285743859:service/1234--web"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
				Body:       `{"cluster":"empire","services":["arn:aws:ecs:us-east-1:249285743859:service/1234--web"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"services":[{"taskDefinition":"1234--web"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"cpu":128,"command":["acme-inc", "web", "--port 80"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web"}]}}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"empire","desiredCount":0,"service":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DeleteService",
				Body:       `{"cluster":"empire","service":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
	})
	m, s := newTestScheduler(h)
	defer s.Close()

	m.lb.(*mockLBManager).On("LoadBalancers", map[string]string{
		"AppID": "1234",
	}).Return([]*lb.LoadBalancer{}, nil)

	if err := m.Remove(context.Background(), "1234"); err != nil {
		t.Fatal(err)
	}
}

func TestScheduler_Processes(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":"empire"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `{"serviceArns":[
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web1",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web2",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web3",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web4",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web5",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web6",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web7",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web8",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web9",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web10",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web11",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web12",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web13",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web14",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web15",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web16",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web17",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web18",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web19",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web20",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web21"
				]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
				Body: `{"cluster":"empire","services":[
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web1",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web2",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web3",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web4",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web5",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web6",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web7",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web8",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web9",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web10"
				]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"services":[{"taskDefinition":"1234--web"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
				Body: `{"cluster":"empire","services":[
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web11",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web12",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web13",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web14",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web15",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web16",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web17",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web18",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web19",
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web20"
				]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"services":[{"taskDefinition":"1234--web"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
				Body: `{"cluster":"empire","services":[
				"arn:aws:ecs:us-east-1:249285743859:service/1234--web21"
				]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"services":[{"taskDefinition":"1234--web"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"cpu":128,"command":["acme-inc", "web", "--port 80"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web"}]}}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"cpu":128,"command":["acme-inc", "web", "--port 80"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web"}]}}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"cpu":128,"command":["acme-inc", "web", "--port 80"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web"}]}}`,
			},
		},
	})
	m, s := newTestScheduler(h)
	defer s.Close()

	if _, err := m.Processes(context.Background(), "1234"); err != nil {
		t.Fatal(err)
	}
}

func TestScheduler_Run(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
				Body:       `{"containerDefinitions":[{"cpu":128,"command":["acme-inc", "web", "--port", "80"],"environment":[{"name":"USER","value":"foo"}],"dockerLabels":{"label1":"foo","label2":"bar"},"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"run"}],"family":"1234--run"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"taskDefinitionArn":"arn:aws:ecs:us-east-1:249285743859:task-definition/1234--run"}}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.RunTask",
				Body:       `{"cluster":"empire","count":1,"startedBy":"1234","taskDefinition":"arn:aws:ecs:us-east-1:249285743859:task-definition/1234--run"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
	})
	m, s := newTestScheduler(h)
	defer s.Close()

	app := &scheduler.App{ID: "1234"}
	process := &scheduler.Process{
		Type:    "run",
		Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
		Command: []string{"acme-inc", "web", "--port", "80"},
		Env: map[string]string{
			"USER": "foo",
		},
		Labels: map[string]string{
			"label1": "foo",
			"label2": "bar",
		},
		MemoryLimit: 134217728, // 128
		CPUShares:   128,
	}
	if err := m.Run(context.Background(), app, process, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDiffProcessTypes(t *testing.T) {
	tests := []struct {
		old, new []*scheduler.Process
		out      []string
	}{
		{nil, nil, []string{}},
		{[]*scheduler.Process{{Type: "web"}}, []*scheduler.Process{{Type: "web"}}, []string{}},
		{[]*scheduler.Process{{Type: "web"}}, nil, []string{"web"}},
		{[]*scheduler.Process{{Type: "web"}, {Type: "worker"}}, []*scheduler.Process{{Type: "web"}}, []string{"worker"}},
	}

	for i, tt := range tests {
		out := diffProcessTypes(tt.old, tt.new)

		if len(out) == 0 && len(tt.out) == 0 {
			continue
		}

		if got, want := out, tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("#%d diffProcessTypes() => %v; want %v", i, got, want)
		}
	}
}

func TestScheduler_LoadBalancer_NoExposure(t *testing.T) {
	l := new(mockLBManager)
	s := &Scheduler{
		lb: l,
	}

	app := &scheduler.App{}
	process := &scheduler.Process{}

	loadBalancer, err := s.loadBalancer(context.Background(), app, process)
	assert.NoError(t, err)
	assert.Nil(t, loadBalancer)

	l.AssertExpectations(t)
}

func TestScheduler_LoadBalancer_NoExistingLoadBalancer(t *testing.T) {
	l := new(mockLBManager)
	s := &Scheduler{
		lb: l,
	}

	app := &scheduler.App{
		ID:   "appid",
		Name: "appname",
	}
	process := &scheduler.Process{
		Type: "web",
		Exposure: &scheduler.Exposure{
			Type: &scheduler.HTTPExposure{},
		},
	}

	l.On("LoadBalancers", map[string]string{"AppID": "appid", "ProcessType": "web"}).Return([]*lb.LoadBalancer{}, nil)
	l.On("CreateLoadBalancer", lb.CreateLoadBalancerOpts{
		Tags:     map[string]string{"AppID": "appid", "ProcessType": "web", "App": "appname"},
		External: false,
	}).Return(&lb.LoadBalancer{
		Name: "lbname",
	}, nil)

	loadBalancer, err := s.loadBalancer(context.Background(), app, process)
	assert.NoError(t, err)
	assert.Equal(t, "lbname", loadBalancer.Name)

	l.AssertExpectations(t)
}

func TestLBProcessManager_CreateProcess_ExistingLoadBalancer(t *testing.T) {
	l := new(mockLBManager)
	s := &Scheduler{
		lb: l,
	}

	app := &scheduler.App{
		ID:   "appid",
		Name: "appname",
	}
	process := &scheduler.Process{
		Type: "web",
		Exposure: &scheduler.Exposure{
			External: true,
			Type:     &scheduler.HTTPExposure{},
		},
	}

	l.On("LoadBalancers", map[string]string{"AppID": "appid", "ProcessType": "web"}).Return([]*lb.LoadBalancer{
		{Name: "lbname", InstancePort: 8080, External: true},
	}, nil)

	loadBalancer, err := s.loadBalancer(context.Background(), app, process)
	assert.NoError(t, err)
	assert.Equal(t, "lbname", loadBalancer.Name)

	l.AssertExpectations(t)
}

func TestScheduler_LoadBalancer_ExistingLoadBalancer_MismatchedExposure(t *testing.T) {
	l := new(mockLBManager)
	s := &Scheduler{
		lb: l,
	}

	app := &scheduler.App{
		ID:   "appid",
		Name: "appname",
	}
	process := &scheduler.Process{
		Type: "web",
		Exposure: &scheduler.Exposure{
			External: true,
			Type:     &scheduler.HTTPExposure{},
		},
	}

	l.On("LoadBalancers", map[string]string{"AppID": "appid", "ProcessType": "web"}).Return([]*lb.LoadBalancer{
		{Name: "lbname", External: false},
	}, nil)

	_, err := s.loadBalancer(context.Background(), app, process)
	assert.EqualError(t, err, "Process web is public, but load balancer is private. An update would require me to delete the load balancer.")

	l.AssertExpectations(t)
}

func TestScheduler_LoadBalancer_ExistingLoadBalancer_NewCert(t *testing.T) {
	l := new(mockLBManager)
	s := &Scheduler{
		lb: l,
	}

	port := int64(8080)
	app := &scheduler.App{
		ID:   "appid",
		Name: "appname",
	}
	process := &scheduler.Process{
		Type: "web",
		Exposure: &scheduler.Exposure{
			External: true,
			Type: &scheduler.HTTPSExposure{
				Cert: "newcert",
			},
		},
	}

	l.On("LoadBalancers", map[string]string{"AppID": "appid", "ProcessType": "web"}).Return([]*lb.LoadBalancer{
		{Name: "lbname", External: true, SSLCert: "oldcert", InstancePort: port},
	}, nil)
	newcert := "newcert"
	l.On("UpdateLoadBalancer", lb.UpdateLoadBalancerOpts{
		Name:    "lbname",
		SSLCert: &newcert,
	}).Return(nil)

	loadBalancer, err := s.loadBalancer(context.Background(), app, process)
	assert.NoError(t, err)
	assert.Equal(t, "lbname", loadBalancer.Name)

	l.AssertExpectations(t)
}

type mockLBManager struct {
	mock.Mock
}

func (m *mockLBManager) CreateLoadBalancer(ctx context.Context, opts lb.CreateLoadBalancerOpts) (*lb.LoadBalancer, error) {
	args := m.Called(opts)
	return args.Get(0).(*lb.LoadBalancer), args.Error(1)
}

func (m *mockLBManager) UpdateLoadBalancer(ctx context.Context, opts lb.UpdateLoadBalancerOpts) error {
	args := m.Called(opts)
	return args.Error(0)
}

func (m *mockLBManager) DestroyLoadBalancer(ctx context.Context, lb *lb.LoadBalancer) error {
	args := m.Called(lb)
	return args.Error(0)
}

func (m *mockLBManager) LoadBalancers(ctx context.Context, tags map[string]string) ([]*lb.LoadBalancer, error) {
	args := m.Called(tags)
	return args.Get(0).([]*lb.LoadBalancer), args.Error(1)
}

// fake app for testing.
var fakeApp = &scheduler.App{
	ID: "1234",
	Processes: []*scheduler.Process{
		&scheduler.Process{
			Type:    "web",
			Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
			Command: []string{"acme-inc", "web", "--port", "80"},
			Env: map[string]string{
				"USER": "foo",
			},
			Labels: map[string]string{
				"label1": "foo",
				"label2": "bar",
			},
			MemoryLimit: 134217728, // 128
			CPUShares:   128,
			Exposure: &scheduler.Exposure{
				Type: &scheduler.HTTPExposure{},
			},
		},
	},
}

func newTestScheduler(h http.Handler) (*Scheduler, *httptest.Server) {
	s := httptest.NewServer(h)

	sched, err := NewScheduler(Config{
		AWS: session.New(&aws.Config{
			Credentials: credentials.NewStaticCredentials(" ", " ", " "),
			Endpoint:    aws.String(s.URL),
			Region:      aws.String("localhost"),
		}),
		Cluster: "empire",
	})
	sched.lb = new(mockLBManager)

	if err != nil {
		panic(err)
	}

	return sched, s
}
