package cloudformation

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	docker "github.com/fsouza/go-dockerclient"
	_ "github.com/lib/pq"
	"github.com/remind101/empire/dbtest"
	"github.com/remind101/empire/internal/uuid"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/twelvefactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	newTimestamp = func() string { return "now" }
}

func TestScheduler_Submit_NewStack(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		ecs:            e,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{}, awserr.New("400", "Stack with id acme-inc does not exist", errors.New(""))).Once()

	c.On("CreateStack", &cloudformation.CreateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		Parameters: []*cloudformation.Parameter{
			{ParameterKey: aws.String("webScale"), ParameterValue: aws.String("1")},
		},
		Tags: []*cloudformation.Tag{
			{Key: aws.String("empire.app.id"), Value: aws.String("c9366591-ab68-4d49-a333-95ce5a23df68")},
			{Key: aws.String("empire.app.name"), Value: aws.String("acme-inc")},
		},
	}).Return(&cloudformation.CreateStackOutput{}, nil)

	c.On("WaitUntilStackCreateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")}},
			},
		},
	}, nil)

	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
		Labels: map[string]string{
			"empire.app.id":   "c9366591-ab68-4d49-a333-95ce5a23df68",
			"empire.app.name": "acme-inc",
		},
		Processes: []*twelvefactor.Process{
			{
				Type:     "web",
				Quantity: 1,
			},
		},
	}, twelvefactor.NullStatusStream)
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_NoDNS(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		ecs:            e,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{}, awserr.New("400", "Stack with id acme-inc does not exist", errors.New(""))).Once()

	c.On("CreateStack", &cloudformation.CreateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		Parameters: []*cloudformation.Parameter{
			{ParameterKey: aws.String("DNS"), ParameterValue: aws.String("false")},
			{ParameterKey: aws.String("webScale"), ParameterValue: aws.String("1")},
		},
		Tags: []*cloudformation.Tag{
			{Key: aws.String("empire.app.id"), Value: aws.String("c9366591-ab68-4d49-a333-95ce5a23df68")},
			{Key: aws.String("empire.app.name"), Value: aws.String("acme-inc")},
		},
	}).Return(&cloudformation.CreateStackOutput{}, nil)

	c.On("WaitUntilStackCreateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")}},
			},
		},
	}, nil)

	err := s.SubmitWithOptions(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
		Labels: map[string]string{
			"empire.app.id":   "c9366591-ab68-4d49-a333-95ce5a23df68",
			"empire.app.name": "acme-inc",
		},
		Processes: []*twelvefactor.Process{
			{
				Type:     "web",
				Quantity: 1,
			},
		},
	}, twelvefactor.NullStatusStream, SubmitOptions{
		NoDNS: aws.Bool(true),
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_ExistingStack(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		ecs:            e,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil).Once()

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{
					&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")},
					&ecs.Deployment{Id: aws.String("2"), Status: aws.String("ACTIVE")},
				},
			},
		},
	}, nil).Once()

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{
					&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")},
				},
			},
		},
	}, nil).Once()

	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
	}, twelvefactor.NullStatusStream)
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_Superseded(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		ecs:            e,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil).Once()

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{
					&ecs.Deployment{Id: aws.String("2"), Status: aws.String("PRIMARY")},
					&ecs.Deployment{Id: aws.String("1"), Status: aws.String("INACTIVE")},
				},
			},
		},
	}, nil)

	stream := &storedStatusStream{}
	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
	}, stream)
	assert.NoError(t, err)
	contains := false
	for _, status := range stream.Statuses() {
		contains = strings.Contains(status.String(), "inactive")
	}
	assert.True(t, contains, "Expected inactive status update")

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_LockWaitTimeout(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		ecs:            e,
		cloudformation: c,
		s3:             x,
		db:             db,
		after: func(d time.Duration) <-chan time.Time {
			if d == lockWait {
				// Return a channel that receives immediately.
				ch := make(chan time.Time)
				close(ch)
				return ch
			}

			return time.After(d)
		},
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil).Once()

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")}},
			},
		},
	}, nil)

	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
	}, twelvefactor.NullStatusStream)
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_StackWaitTimeout(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		ecs:            e,
		s3:             x,
		db:             db,
		after: func(d time.Duration) <-chan time.Time {
			if d == stackOperationTimeout {
				// Return a channel that receives immediately.
				ch := make(chan time.Time)
				close(ch)
				return ch
			}

			return time.After(d)
		},
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil).Once()

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")}},
			},
		},
	}, nil)

	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
	}, twelvefactor.NullStatusStream)
	assert.EqualError(t, err, `timed out waiting for stack operation to complete`)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_UpdateError(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		ecs:            e,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil).Once()

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, errors.New("stack update failed"))

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")}},
			},
		},
	}, nil)

	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
	}, twelvefactor.NullStatusStream)
	assert.EqualError(t, err, `error updating stack: stack update failed`)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_ExistingStack_RemovedProcess(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		ecs:            e,
		cloudformation: c,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{
		Parameters: []*cloudformation.TemplateParameter{
			{ParameterKey: aws.String("webScale")},
		},
	}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Parameters: []*cloudformation.Parameter{
					{ParameterKey: aws.String("webScale"), ParameterValue: aws.String("1")},
					{ParameterKey: aws.String("workerScale"), ParameterValue: aws.String("0")},
				},
			},
		},
	}, nil).Once()

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		Parameters: []*cloudformation.Parameter{
			{ParameterKey: aws.String("webScale"), ParameterValue: aws.String("1")},
		},
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")}},
			},
		},
	}, nil)

	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
		Processes: []*twelvefactor.Process{
			{Type: "web", Quantity: 1},
		},
	}, twelvefactor.NullStatusStream)
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_ExistingStack_ExistingParameterValue(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		ecs:            e,
		cloudformation: c,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{
		Parameters: []*cloudformation.TemplateParameter{
			{ParameterKey: aws.String("DNS")},
			{ParameterKey: aws.String("webScale")},
		},
	}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Parameters: []*cloudformation.Parameter{
					{ParameterKey: aws.String("DNS"), ParameterValue: aws.String("false")},
					{ParameterKey: aws.String("webScale"), ParameterValue: aws.String("1")},
					{ParameterKey: aws.String("workerScale"), ParameterValue: aws.String("0")},
				},
			},
		},
	}, nil).Once()

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Parameters: []*cloudformation.Parameter{
					{ParameterKey: aws.String("DNS"), ParameterValue: aws.String("false")},
					{ParameterKey: aws.String("webScale"), ParameterValue: aws.String("1")},
					{ParameterKey: aws.String("workerScale"), ParameterValue: aws.String("0")},
				},
			},
		},
	}, nil).Once()

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		Parameters: []*cloudformation.Parameter{
			{ParameterKey: aws.String("webScale"), ParameterValue: aws.String("1")},
			{ParameterKey: aws.String("DNS"), UsePreviousValue: aws.Bool(true)},
		},
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("Deployments"),
						OutputValue: aws.String("web=1"),
					},
				},
			},
		},
	}, nil)

	e.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
				Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("1"), Status: aws.String("PRIMARY")}},
			},
		},
	}, nil)

	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
		Processes: []*twelvefactor.Process{
			{Type: "web", Quantity: 1},
		},
	}, twelvefactor.NullStatusStream)
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_TemplateTooLarge(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("ValidateTemplate", &cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.ValidateTemplateOutput{}, awserr.New("ValidationError", "Template may not exceed 460800 bytes in size.", errors.New("")))

	err := s.Submit(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
	}, twelvefactor.NullStatusStream)
	assert.EqualError(t, err, `TemplateValidationError:
  Template URL: https://bucket.s3.amazonaws.com/acme-inc/c9366591-ab68-4d49-a333-95ce5a23df68/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f
  Template Size: 2 bytes
  Error: ValidationError: Template may not exceed 460800 bytes in size.
caused by: `)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Remove(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	c.On("DeleteStack", &cloudformation.DeleteStackInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DeleteStackOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
				},
			},
		},
	}, nil)

	err = s.Remove(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68")
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Remove_NoCFStack(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{}, awserr.New("400", "Stack with id acme-inc does not exist", errors.New("")))

	err = s.Remove(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68")
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Remove_NoDBStack_NoCFStack(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	err := s.Remove(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68")
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Instances(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		s3:             x,
		ecs:            e,
		db:             db,
		after:          fakeAfter,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
				},
			},
		},
	}, nil)

	e.On("ListTasksPages", &ecs.ListTasksInput{
		Cluster:     aws.String("cluster"),
		ServiceName: aws.String("acme-inc-web"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
		},
	}, nil)

	e.On("ListTasksPages", &ecs.ListTasksInput{
		Cluster:   aws.String("cluster"),
		StartedBy: aws.String("c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
		},
	}, nil)

	dt := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	e.On("DescribeTasks", &ecs.DescribeTasksInput{
		Cluster: aws.String("cluster"),
		Tasks: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
		},
	}).Return(&ecs.DescribeTasksOutput{
		Tasks: []*ecs.Task{
			{
				TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
				TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-web:0"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-1"),
				ClusterArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster-name-1"),
				LastStatus:           aws.String("RUNNING"),
				StartedAt:            &dt,
			},
			{
				TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
				TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc--run:0"),
				ClusterArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster-name-2"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-2"),
				LastStatus:           aws.String("PENDING"),
				CreatedAt:            &dt,
			},
		},
	}, nil)

	e.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-web:0"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name:   aws.String("web"),
					Cpu:    aws.Int64(256),
					Memory: aws.Int64(int64(256)),
				},
			},
		},
	}, nil)

	e.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc--run:0"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name:   aws.String("run"),
					Cpu:    aws.Int64(256),
					Memory: aws.Int64(int64(256)),
				},
			},
		},
	}, nil)

	e.On("DescribeContainerInstances", &ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster-name-1"),
		ContainerInstances: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-1")},
	}).Return(&ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []*ecs.ContainerInstance{
			{
				Ec2InstanceId:        aws.String("ec2-instance-id-1"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-1"),
			},
		},
	}, nil)

	e.On("DescribeContainerInstances", &ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster-name-2"),
		ContainerInstances: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-2")},
	}).Return(&ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []*ecs.ContainerInstance{
			{
				Ec2InstanceId:        aws.String("ec2-instance-id-2"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-2"),
			},
		},
	}, nil)

	instances, err := s.Tasks(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68")
	assert.NoError(t, err)
	assert.Equal(t, &twelvefactor.Task{
		ID:        "0b69d5c0-d655-4695-98cd-5d2d526d9d5a",
		Host:      twelvefactor.Host{ID: "ec2-instance-id-1"},
		UpdatedAt: dt,
		State:     "RUNNING",
		Process: &twelvefactor.Process{
			Type:      "web",
			Memory:    256 * bytesize.MB,
			CPUShares: 256,
			Env:       make(map[string]string),
		},
	}, instances[0])
	assert.Equal(t, &twelvefactor.Task{
		ID:        "c09f0188-7f87-4b0f-bfc3-16296622b6fe",
		Host:      twelvefactor.Host{ID: "ec2-instance-id-2"},
		UpdatedAt: dt,
		State:     "PENDING",
		Process: &twelvefactor.Process{
			Type:      "run",
			Memory:    256 * bytesize.MB,
			CPUShares: 256,
			Env:       make(map[string]string),
		},
	}, instances[1])

	c.AssertExpectations(t)
	x.AssertExpectations(t)
	e.AssertExpectations(t)
}

func TestScheduler_Instances_ManyTasks(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		Cluster:        "cluster",
		cloudformation: c,
		s3:             x,
		ecs:            e,
		db:             db,
		after:          fakeAfter,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
				},
			},
		},
	}, nil)

	var page1 []*string
	for i := 0; i < MaxDescribeTasks; i++ {
		arn := fmt.Sprintf("arn:aws:ecs:us-east-1:012345678910:task/%s", uuid.New())
		page1 = append(page1, aws.String(arn))
	}
	page2 := []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe")}
	e.On("ListTasksPages", &ecs.ListTasksInput{
		Cluster:     aws.String("cluster"),
		ServiceName: aws.String("acme-inc-web"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: append(page1, page2...),
	}, nil)

	e.On("ListTasksPages", &ecs.ListTasksInput{
		Cluster:   aws.String("cluster"),
		StartedBy: aws.String("c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: []*string{},
	}, nil)

	e.On("DescribeTasks", &ecs.DescribeTasksInput{
		Cluster: aws.String("cluster"),
		Tasks:   page1,
	}).Return(&ecs.DescribeTasksOutput{
		Tasks: []*ecs.Task{
		// In reality, this would return all the tasks, but we
		// just want to test that task arns are chunked
		// properly.
		},
	}, nil)

	e.On("DescribeTasks", &ecs.DescribeTasksInput{
		Cluster: aws.String("cluster"),
		Tasks:   page2,
	}).Return(&ecs.DescribeTasksOutput{
		Tasks: []*ecs.Task{
		// In reality, this would return all the tasks, but we
		// just want to test that task arns are chunked
		// properly.
		},
	}, nil)

	_, err = s.Tasks(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68")
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
	e.AssertExpectations(t)
}

func TextExtractProcessData(t *testing.T) {
	output := "statuses=arn:aws:ecs:us-east-1:897883143566:service/stage-app-statuses-16NM105QFD6UO,statuses_retry=arn:aws:ecs:us-east-1:897883143566:service/stage-app-statusesretry-DKG2XMH75H5N"
	services := extractProcessData(output)
	expected := map[string]string{
		"statuses":       "arn:aws:ecs:us-east-1:897883143566:service/stage-app-statuses-16NM105QFD6UO",
		"statuses_retry": "arn:aws:ecs:us-east-1:897883143566:service/stage-app-statusesretry-DKG2XMH75H5N",
	}

	assert.Equal(t, expected, services)
}

func TestChunkStrings(t *testing.T) {
	tests := []struct {
		in  []*string
		out [][]*string
	}{
		{[]*string{aws.String("a")}, [][]*string{[]*string{aws.String("a")}}},
		{[]*string{aws.String("a"), aws.String("b")}, [][]*string{[]*string{aws.String("a"), aws.String("b")}}},
		{[]*string{aws.String("a"), aws.String("b"), aws.String("c")}, [][]*string{[]*string{aws.String("a"), aws.String("b")}, []*string{aws.String("c")}}},
		{[]*string{aws.String("a"), aws.String("b"), aws.String("c"), aws.String("d")}, [][]*string{[]*string{aws.String("a"), aws.String("b")}, []*string{aws.String("c"), aws.String("d")}}},
	}

	for _, tt := range tests {
		out := chunkStrings(tt.in, 2)
		assert.Equal(t, tt.out, out)
	}
}

func TestUpdateParameters(t *testing.T) {
	tests := []struct {
		parameters []*cloudformation.Parameter
		stack      *cloudformation.Stack
		template   *cloudformationTemplate

		out []*cloudformation.Parameter
	}{
		// Simple scenario, overwriting a parameter value with a new
		// value.
		{
			[]*cloudformation.Parameter{
				{ParameterKey: aws.String("a"), ParameterValue: aws.String("false")},
			},
			&cloudformation.Stack{
				Parameters: []*cloudformation.Parameter{
					{ParameterKey: aws.String("a"), ParameterValue: aws.String("true")},
					{ParameterKey: aws.String("b"), ParameterValue: aws.String("false")},
				},
			},
			nil,
			[]*cloudformation.Parameter{
				{ParameterKey: aws.String("a"), ParameterValue: aws.String("false")},
				{ParameterKey: aws.String("b"), UsePreviousValue: aws.Bool(true)},
			},
		},

		// Updating with a new template, that doesn't provide one of the
		// values.
		{
			[]*cloudformation.Parameter{
				{ParameterKey: aws.String("a"), ParameterValue: aws.String("false")},
			},
			&cloudformation.Stack{
				Parameters: []*cloudformation.Parameter{
					{ParameterKey: aws.String("a"), ParameterValue: aws.String("true")},
					{ParameterKey: aws.String("b"), ParameterValue: aws.String("false")},
				},
			},
			&cloudformationTemplate{
				Parameters: []*cloudformation.TemplateParameter{
					{ParameterKey: aws.String("a")},
				},
			},
			[]*cloudformation.Parameter{
				{ParameterKey: aws.String("a"), ParameterValue: aws.String("false")},
			},
		},

		// Updating a stack that has a new parameter, but setting it to
		// the default.
		{
			[]*cloudformation.Parameter{
				{ParameterKey: aws.String("a"), ParameterValue: aws.String("false")},
			},
			&cloudformation.Stack{
				Parameters: []*cloudformation.Parameter{
					{ParameterKey: aws.String("a"), ParameterValue: aws.String("true")},
				},
			},
			&cloudformationTemplate{
				Parameters: []*cloudformation.TemplateParameter{
					{ParameterKey: aws.String("a")},
					{ParameterKey: aws.String("b")},
				},
			},
			[]*cloudformation.Parameter{
				{ParameterKey: aws.String("a"), ParameterValue: aws.String("false")},
			},
		},
	}

	for _, tt := range tests {
		out := updateParameters(tt.parameters, tt.stack, tt.template)
		assert.Equal(t, tt.out, out)
	}
}

func TestScheduler_Restart(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
		after:          fakeAfter,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil)

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:           aws.String("acme-inc"),
		UsePreviousTemplate: aws.Bool(true),
		Parameters: []*cloudformation.Parameter{
			{ParameterKey: aws.String("RestartKey"), ParameterValue: aws.String("now")},
		},
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	err = s.Restart(context.Background(), "c9366591-ab68-4d49-a333-95ce5a23df68", twelvefactor.NullStatusStream)
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Run_Detached(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	e := new(mockECSClient)
	s := &Scheduler{
		Template: &fakeTemplate{
			containerDefinition: &ecs.ContainerDefinition{},
		},
		ecs:   e,
		db:    db,
		after: fakeAfter,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	e.On("RegisterTaskDefinition", &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String("c9366591-ab68-4d49-a333-95ce5a23df68--run"),
		TaskRoleArn: aws.String("arn:aws:iam::897883143566:role/app"),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{},
		},
	}).Return(&ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/c9366591-ab68-4d49-a333-95ce5a23df68--run:0"),
		},
	}, nil)

	e.On("RunTask", &ecs.RunTaskInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/c9366591-ab68-4d49-a333-95ce5a23df68--run:0"),
		Cluster:        aws.String(""),
		Count:          aws.Int64(1),
		StartedBy:      aws.String("c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(&ecs.RunTaskOutput{
		Tasks: []*ecs.Task{
			&ecs.Task{
				ClusterArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
				DesiredStatus:        aws.String("RUNNING"),
				LastStatus:           aws.String("PENDING"),
				TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb"),
				TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/c9366591-ab68-4d49-a333-95ce5a23df68--run:0"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/4c543eed-f83f-47da-b1d8-3d23f1da4c64"),
			},
		},
	}, nil)

	err = s.Run(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
		Env: map[string]string{
			"EMPIRE_X_TASK_ROLE_ARN": "arn:aws:iam::897883143566:role/app",
		},
		Processes: []*twelvefactor.Process{
			&twelvefactor.Process{
				Type:    "run",
				Command: []string{"bundle exec rake db:migrate"},
			},
		},
	})
	assert.NoError(t, err)

	e.AssertExpectations(t)
}

func TestScheduler_Run_Attached(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	e := new(mockECSClient)
	c := new(mockEC2Client)
	d := new(mockDockerClient)
	s := &Scheduler{
		Template: &fakeTemplate{
			containerDefinition: &ecs.ContainerDefinition{
				Command: []*string{aws.String("bundle"), aws.String("exec"), aws.String("rake"), aws.String("db:migrate")},
			},
		},
		NewDockerClient: func(ec2Instance *ec2.Instance) (DockerClient, error) {
			return d, nil
		},
		ecs:   e,
		ec2:   c,
		db:    db,
		after: fakeAfter,
	}

	_, err := db.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, "c9366591-ab68-4d49-a333-95ce5a23df68", "acme-inc")
	assert.NoError(t, err)

	e.On("RegisterTaskDefinition", &ecs.RegisterTaskDefinitionInput{
		Family: aws.String("c9366591-ab68-4d49-a333-95ce5a23df68--run"),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{
				Command: []*string{aws.String("bundle"), aws.String("exec"), aws.String("rake"), aws.String("db:migrate")},
				DockerLabels: map[string]*string{
					"docker.config.Tty":       aws.String("true"),
					"docker.config.OpenStdin": aws.String("true"),
				},
			},
		},
	}).Return(&ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/c9366591-ab68-4d49-a333-95ce5a23df68--run:0"),
		},
	}, nil)

	e.On("RunTask", &ecs.RunTaskInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/c9366591-ab68-4d49-a333-95ce5a23df68--run:0"),
		Cluster:        aws.String(""),
		Count:          aws.Int64(1),
		StartedBy:      aws.String("c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(&ecs.RunTaskOutput{
		Tasks: []*ecs.Task{
			&ecs.Task{
				ClusterArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
				DesiredStatus:        aws.String("RUNNING"),
				LastStatus:           aws.String("PENDING"),
				TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb"),
				TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/c9366591-ab68-4d49-a333-95ce5a23df68--run:0"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/4c543eed-f83f-47da-b1d8-3d23f1da4c64"),
			},
		},
	}, nil)

	e.On("StopTask", &ecs.StopTaskInput{
		Cluster: aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
		Task:    aws.String("arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb"),
	}).Return(&ecs.StopTaskOutput{}, nil)

	e.On("DescribeContainerInstances", &ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
		ContainerInstances: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/4c543eed-f83f-47da-b1d8-3d23f1da4c64")},
	}).Return(&ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []*ecs.ContainerInstance{
			&ecs.ContainerInstance{
				AgentConnected:       aws.Bool(true),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/4c543eed-f83f-47da-b1d8-3d23f1da4c64"),
				Ec2InstanceId:        aws.String("i-042f39dc"),
			},
		},
	}, nil)

	c.On("DescribeInstances", &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String("i-042f39dc")},
	}).Return(&ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			&ec2.Reservation{
				Instances: []*ec2.Instance{
					&ec2.Instance{
						InstanceId:       aws.String("i-042f39dc"),
						PrivateIpAddress: aws.String("192.168.1.88"),
					},
				},
			},
		},
	}, nil)

	e.On("WaitUntilTasksNotPending", &ecs.DescribeTasksInput{
		Cluster: aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
		Tasks:   []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb")},
	}).Return(nil)

	d.On("ListContainers", docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label": []string{"com.amazonaws.ecs.task-arn=arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb"},
		},
	}).Return([]docker.APIContainers{
		docker.APIContainers{
			ID: "4c01db0b339c",
		},
	}, nil)

	stdin := strings.NewReader("ls\n")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	d.On("AttachToContainer", docker.AttachToContainerOptions{
		Container:    "4c01db0b339c",
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
		Logs:         true,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	}).Return(nil)

	err = s.Run(context.Background(), &twelvefactor.Manifest{
		AppID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name:  "acme-inc",
		Processes: []*twelvefactor.Process{
			&twelvefactor.Process{
				Type:    "run",
				Command: []string{"bundle", "exec", "rake", "db:migrate"},
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
			},
		},
	})
	assert.NoError(t, err)

	assert.Equal(t, "Attaching to task/fdf2c302-468c-4e55-b884-5331d816e7fb...\r\n", stderr.String())

	e.AssertExpectations(t)
	c.AssertExpectations(t)
	d.AssertExpectations(t)
}

func newDB(t testing.TB) *sql.DB {
	db := dbtest.Open(t)
	if _, err := db.Exec(`TRUNCATE TABLE stacks`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`TRUNCATE TABLE scheduler_migration`); err != nil {
		t.Fatal(err)
	}
	return db
}

type fakeTemplate struct {
	template            string
	containerDefinition *ecs.ContainerDefinition
}

func (t *fakeTemplate) Execute(wr io.Writer, data interface{}) error {
	tmpl := t.template
	if tmpl == "" {
		tmpl = "{}"
	}
	_, err := io.WriteString(wr, tmpl)
	return err
}

func (t *fakeTemplate) ContainerDefinition(*twelvefactor.Manifest, *twelvefactor.Process) *ecs.ContainerDefinition {
	return t.containerDefinition
}

type storedStatusStream struct {
	sync.Mutex
	statuses []twelvefactor.Status
}

func (s *storedStatusStream) Publish(status twelvefactor.Status) error {
	s.Lock()
	defer s.Unlock()
	s.statuses = append(s.statuses, status)
	return nil
}

func (s *storedStatusStream) Statuses() []twelvefactor.Status {
	s.Lock()
	defer s.Unlock()
	return s.statuses
}

type mockCloudFormationClient struct {
	cloudformationClient
	mock.Mock
}

func (m *mockCloudFormationClient) CreateStack(input *cloudformation.CreateStackInput) (*cloudformation.CreateStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.CreateStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) UpdateStack(input *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.UpdateStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) DeleteStack(input *cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DeleteStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) DescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DescribeStacksOutput), args.Error(1)
}

func (m *mockCloudFormationClient) ListStackResourcesPages(input *cloudformation.ListStackResourcesInput, fn func(*cloudformation.ListStackResourcesOutput, bool) bool) error {
	args := m.Called(input)
	fn(args.Get(0).(*cloudformation.ListStackResourcesOutput), true)
	return args.Error(1)
}

func (m *mockCloudFormationClient) DescribeStackResource(input *cloudformation.DescribeStackResourceInput) (*cloudformation.DescribeStackResourceOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DescribeStackResourceOutput), args.Error(1)
}

func (m *mockCloudFormationClient) WaitUntilStackCreateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

func (m *mockCloudFormationClient) WaitUntilStackUpdateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

func (m *mockCloudFormationClient) ValidateTemplate(input *cloudformation.ValidateTemplateInput) (*cloudformation.ValidateTemplateOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.ValidateTemplateOutput), args.Error(1)
}

type mockS3Client struct {
	mock.Mock
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

type mockECSClient struct {
	ecsClient
	mock.Mock
}

func (m *mockECSClient) ListTasksPages(input *ecs.ListTasksInput, fn func(p *ecs.ListTasksOutput, lastPage bool) (shouldContinue bool)) error {
	args := m.Called(input)
	fn(args.Get(0).(*ecs.ListTasksOutput), true)
	return args.Error(1)
}

func (m *mockECSClient) DescribeTasks(input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeTasksOutput), args.Error(1)
}

func (m *mockECSClient) DescribeTaskDefinition(input *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeTaskDefinitionOutput), args.Error(1)
}

func (m *mockECSClient) DescribeServices(input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeServicesOutput), args.Error(1)
}

func (m *mockECSClient) DescribeContainerInstances(input *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeContainerInstancesOutput), args.Error(1)
}

func (m *mockECSClient) RegisterTaskDefinition(input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.RegisterTaskDefinitionOutput), args.Error(1)
}

func (m *mockECSClient) RunTask(input *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.RunTaskOutput), args.Error(1)
}

func (m *mockECSClient) StopTask(input *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.StopTaskOutput), args.Error(1)
}

func (m *mockECSClient) WaitUntilTasksNotPending(input *ecs.DescribeTasksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

type mockEC2Client struct {
	mock.Mock
}

func (m *mockEC2Client) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ec2.DescribeInstancesOutput), args.Error(1)
}

type mockDockerClient struct {
	mock.Mock
}

func (m *mockDockerClient) ListContainers(options docker.ListContainersOptions) ([]docker.APIContainers, error) {
	args := m.Called(options)
	return args.Get(0).([]docker.APIContainers), args.Error(1)
}

func (m *mockDockerClient) AttachToContainer(options docker.AttachToContainerOptions) error {
	args := m.Called(options)
	return args.Error(0)
}

// fakeAfter is a helper function that will resolve immediately
// except in cases where a lockWait is specified.
func fakeAfter(d time.Duration) <-chan time.Time {
	if d == lockWait || d == stackOperationTimeout {
		return nil
	}
	ch := make(chan time.Time)
	close(ch)
	return ch
}
