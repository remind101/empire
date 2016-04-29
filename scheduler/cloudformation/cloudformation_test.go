package cloudformation

import (
	"bytes"
	"database/sql"
	"errors"
	"html/template"
	"testing"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/s3"
	_ "github.com/lib/pq"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestScheduler_Submit_NewStack(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{}, awserr.New("400", "Stack with id acme-inc does not exist", errors.New("")))

	c.On("CreateStack", &cloudformation.CreateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		Tags: []*cloudformation.Tag{
			{Key: aws.String("empire.app.id"), Value: aws.String("c9366591-ab68-4d49-a333-95ce5a23df68")},
			{Key: aws.String("empire.app.name"), Value: aws.String("acme-inc")},
		},
	}).Return(&cloudformation.CreateStackOutput{}, nil)

	c.On("WaitUntilStackCreateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	err := s.Submit(context.Background(), &scheduler.App{
		ID:   "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name: "acme-inc",
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
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil)

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	err := s.Submit(context.Background(), &scheduler.App{
		ID:   "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func TestScheduler_Submit_StackUpdateInProgress(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	x := new(mockS3Client)
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		Bucket:         "bucket",
		cloudformation: c,
		s3:             x,
		db:             db,
	}

	x.On("PutObject", &s3.PutObjectInput{
		Bucket:      aws.String("bucket"),
		Body:        bytes.NewReader([]byte("{}")),
		Key:         aws.String("/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
		ContentType: aws.String("application/json"),
	}).Return(&s3.PutObjectOutput{}, nil)

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("UPDATE_IN_PROGRESS")},
		},
	}, nil)

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:   aws.String("acme-inc"),
		TemplateURL: aws.String("https://bucket.s3.amazonaws.com/bf21a9e8fbc5a3846fb05b4fa0859e0917b2202f"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil).Twice()

	err := s.Submit(context.Background(), &scheduler.App{
		ID:   "c9366591-ab68-4d49-a333-95ce5a23df68",
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
	x.AssertExpectations(t)
}

func newDB(t testing.TB) *sql.DB {
	db, err := sql.Open("postgres", "postgres://localhost/empire?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`TRUNCATE TABLE stacks`); err != nil {
		t.Fatal(err)
	}
	return db
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

func (m *mockCloudFormationClient) DescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DescribeStacksOutput), args.Error(1)
}

func (m *mockCloudFormationClient) WaitUntilStackCreateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

func (m *mockCloudFormationClient) WaitUntilStackUpdateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

type mockS3Client struct {
	mock.Mock
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}
