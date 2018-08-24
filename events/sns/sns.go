// Package sns provides an empire.EventStream implementation that publishes events to
// SNS.
package sns

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/remind101/empire"
)

// Event represents the schema for an SNS payload.
type Event struct {
	Event   string
	Message string
	User    string
	Data    interface{}
}

type snsClient interface {
	Publish(*sns.PublishInput) (*sns.PublishOutput, error)
}

// EventStream is an implementation of the empire.EventStream interface backed
// by SNS.
type EventStream struct {
	// The topic to publish events to.
	TopicARN string

	sns snsClient
}

func NewEventStream(c client.ConfigProvider) *EventStream {
	return &EventStream{
		sns: sns.New(c),
	}
}

func (e *EventStream) PublishEvent(event empire.Event) error {
	raw, err := json.Marshal(&Event{
		Event:   event.Event(),
		Message: event.String(),
		User:    event.User().Name,
		Data:    event,
	})
	if err != nil {
		return err
	}

	_, err = e.sns.Publish(&sns.PublishInput{
		Message:  aws.String(string(raw)),
		TopicArn: aws.String(e.TopicARN),
	})
	return err
}
