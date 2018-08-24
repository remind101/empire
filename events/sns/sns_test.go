package sns

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEvents_PublishEvent(t *testing.T) {
	c := new(mockSNSClient)
	e := &EventStream{
		TopicARN: "arn::sns/topic",
		sns:      c,
	}

	c.On("Publish", &sns.PublishInput{
		Message:  aws.String(`{"Event":"fake","Message":"ejholmes did something","User":"ejholmes","Data":{}}`),
		TopicArn: aws.String("arn::sns/topic"),
	}).Return(nil, nil)

	err := e.PublishEvent(fakeEvent{
		user: &empire.User{
			Name: "ejholmes",
		},
	})
	assert.NoError(t, err)
}

type fakeEvent struct {
	user *empire.User
}

func (e fakeEvent) Event() string      { return "fake" }
func (e fakeEvent) String() string     { return fmt.Sprintf("%s did something", e.User()) }
func (e fakeEvent) Message() string    { return "message" }
func (e fakeEvent) User() *empire.User { return e.user }

type mockSNSClient struct {
	mock.Mock
}

func (m *mockSNSClient) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	args := m.Called(input)
	return nil, args.Error(1)
}
