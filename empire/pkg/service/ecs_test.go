package service

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/remind101/empire/empire/pkg/awsutil"
	"golang.org/x/net/context"
)

func TestECSManager_Submit(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":""}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":["arn:aws:ecs:us-east-1:249285743859:service/foo--web"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
				Body:       `{"cluster":"","services":["arn:aws:ecs:us-east-1:249285743859:service/foo--web"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"services":[{"taskDefinition":"foo--web"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"cpu":128,"command":["acme-inc","web"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web"}]}}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
				Body:       `{"containerDefinitions":[{"cpu":128,"command":["acme-inc","web"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web","portMappings":[{"containerPort":8080,"hostPort":8080}]}],"family":"foo--web"}`,
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
				Body:       `{"cluster":"","desiredCount":0,"service":"foo--web","taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"service": {}}`,
			},
		},
	})
	m, s := newTestECSManager(h)
	defer s.Close()

	if err := m.Submit(context.Background(), fakeApp); err != nil {
		t.Fatal(err)
	}
}

func TestECSManager_Scale(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"","desiredCount":10,"service":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
	})
	m, s := newTestECSManager(h)
	defer s.Close()

	if err := m.Scale(context.Background(), "foo", "web", 10); err != nil {
		t.Fatal(err)
	}
}

func TestECSManager_Instances(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":""}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":["arn:aws:ecs:us-east-1:249285743859:service/foo--web"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
				Body:       `{"cluster":"","serviceName":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskArns":["arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
				Body:       `{"tasks":["arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"tasks":[{"taskArn":"arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570","taskDefinitionArn":"arn:aws:ecs:us-east-1:249285743859:task-definition/foo--web","lastStatus":"RUNNING"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"arn:aws:ecs:us-east-1:249285743859:task-definition/foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"name":"web","command":["acme-inc", "web"]}]}}`,
			},
		},
	})
	m, s := newTestECSManager(h)
	defer s.Close()

	instances, err := m.Instances(context.Background(), "foo")
	if err != nil {
		t.Fatal(err)
	}

	if len(instances) != 1 {
		t.Fatal("expected 1 instance")
	}

	i := instances[0]

	if got, want := i.State, "RUNNING"; got != want {
		t.Fatalf("State => %s; want %s", got, want)
	}

	if got, want := i.Process.Command, "acme-inc web"; got != want {
		t.Fatalf("Command => %s; want %s", got, want)
	}

	if got, want := i.Process.Type, "web"; got != want {
		t.Fatalf("Type => %s; want %s", got, want)
	}
}

func TestECSManager_Remove(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":""}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":["arn:aws:ecs:us-east-1:249285743859:service/foo--web"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
				Body:       `{"cluster":"","services":["arn:aws:ecs:us-east-1:249285743859:service/foo--web"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"services":[{"taskDefinition":"foo--web"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"cpu":128,"command":["acme-inc","web"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web"}]}}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"","desiredCount":0,"service":"foo--web"}`,
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
				Body:       `{"cluster":"","service":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
	})
	m, s := newTestECSManager(h)
	defer s.Close()

	if err := m.Remove(context.Background(), "foo"); err != nil {
		t.Fatal(err)
	}
}

func TestDiffProcessTypes(t *testing.T) {
	tests := []struct {
		old, new []*Process
		out      []string
	}{
		{nil, nil, []string{}},
		{[]*Process{{Type: "web"}}, []*Process{{Type: "web"}}, []string{}},
		{[]*Process{{Type: "web"}}, nil, []string{"web"}},
		{[]*Process{{Type: "web"}, {Type: "worker"}}, []*Process{{Type: "web"}}, []string{"worker"}},
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

// fake app for testing.
var fakeApp = &App{
	Name: "foo",
	Processes: []*Process{
		&Process{
			Type:    "web",
			Image:   "remind101/acme-inc:latest",
			Command: "acme-inc web",
			Env: map[string]string{
				"USER": "foo",
			},
			MemoryLimit: 134217728, // 128
			CPUShares:   128,
			Ports: []PortMap{
				{aws.Long(8080), aws.Long(8080)},
			},
			Exposure: ExposePrivate,
		},
	},
}

func newTestECSManager(h http.Handler) (*ECSManager, *httptest.Server) {
	s := httptest.NewServer(h)

	return NewECSManager(
		aws.DefaultConfig.Merge(&aws.Config{
			Credentials: aws.Creds("", "", ""),
			Endpoint:    s.URL,
			Region:      "localhost",
		}),
	), s
}
