package scheduler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/remind101/empire/pkg/awsutil"
	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

func TestECSManager_Submit(t *testing.T) {
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
				Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
				Body:       `{"containerDefinitions":[{"cpu":128,"command":["acme-inc", "web", "--port 80"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web","portMappings":[{"containerPort":8080,"hostPort":8080}]}],"family":"1234--web"}`,
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
				Body:       `{"cluster":"empire","desiredCount":10,"service":"1234--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
	})
	m, s := newTestECSManager(h)
	defer s.Close()

	if err := m.Scale(context.Background(), "1234", "web", 10); err != nil {
		t.Fatal(err)
	}
}

func TestECSManager_Instances(t *testing.T) {
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
				Body:       `{"taskArns":["arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
				Body:       `{"cluster":"empire","tasks":["arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"tasks":[{"taskArn":"arn:aws:ecs:us-east-1:249285743859:task/ae69bb4c-3903-4844-82fe-548ac5b74570","taskDefinitionArn":"arn:aws:ecs:us-east-1:249285743859:task-definition/1234--web","lastStatus":"RUNNING"}]}`,
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
				Body:       `{"taskDefinition":{"containerDefinitions":[{"name":"web","cpu":256,"memory":256,"command":["acme-inc", "web", "--port 80"]}]}}`,
			},
		},
	})
	m, s := newTestECSManager(h)
	defer s.Close()

	instances, err := m.Instances(context.Background(), "1234")
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

	if got, want := i.Process.Command, "acme-inc web --port 80"; got != want {
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
	m, s := newTestECSManager(h)
	defer s.Close()

	if err := m.Remove(context.Background(), "1234"); err != nil {
		t.Fatal(err)
	}
}

func TestECSManager_Processes(t *testing.T) {
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
	m, s := newTestECSManager(h)
	defer s.Close()

	if _, err := m.Processes(context.Background(), "1234"); err != nil {
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
	ID: "1234",
	Processes: []*Process{
		&Process{
			Type:    "web",
			Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
			Command: "acme-inc web '--port 80'",
			Env: map[string]string{
				"USER": "foo",
			},
			MemoryLimit: 134217728, // 128
			CPUShares:   128,
			Ports: []PortMap{
				{aws.Int64(8080), aws.Int64(8080)},
			},
			Exposure: ExposePrivate,
		},
	},
}

func newTestECSManager(h http.Handler) (*ECSManager, *httptest.Server) {
	s := httptest.NewServer(h)

	m, err := NewECSManager(ECSConfig{
		AWS: aws.NewConfig().Merge(&aws.Config{
			Credentials: credentials.NewStaticCredentials(" ", " ", " "),
			Endpoint:    aws.String(s.URL),
			Region:      aws.String("localhost"),
		}),
		Cluster: "empire",
	})

	if err != nil {
		panic(err)
	}

	return m, s
}
