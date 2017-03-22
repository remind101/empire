package cloudwatch

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEvents_PublishEvent(t *testing.T) {
	c := new(mockCloudWatchEventsClient)
	e := &EventStream{
		Environment: "staging",
		client:      c,
	}

	c.On("PutEvents", &cloudwatchevents.PutEventsInput{
		Entries: []*cloudwatchevents.PutEventsRequestEntry{
			{
				DetailType: aws.String("fake"),
				Source:     aws.String("empire"),
				Detail:     aws.String("{\"Environment\":\"staging\",\"Event\":\"fake\",\"Message\":\"ejholmes did something\",\"Data\":{\"User\":\"ejholmes\"}}"),
			},
		},
	}).Return(nil, nil)

	err := e.PublishEvent(fakeEvent{
		User: "ejholmes",
	})
	assert.NoError(t, err)
}

type fakeEvent struct {
	User string
}

func (e fakeEvent) Event() string  { return "fake" }
func (e fakeEvent) String() string { return fmt.Sprintf("%s did something", e.User) }

type mockCloudWatchEventsClient struct {
	mock.Mock
}

func (m *mockCloudWatchEventsClient) PutEvents(input *cloudwatchevents.PutEventsInput) (*cloudwatchevents.PutEventsOutput, error) {
	args := m.Called(input)
	return nil, args.Error(1)
}
