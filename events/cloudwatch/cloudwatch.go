// Package cloudwatch provides an empire.EventStream implementation that publishes events to
// CloudWatch Events.
package cloudwatch

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/remind101/empire"
)

// Maps to the `source` key of a CloudWatch Event.
const Source = "empire"

// Event represents the schema for the Detail field.
type Event struct {
	Environment string
	Event       string
	Message     string
	Data        interface{}
}

type cloudwatcheventsClient interface {
	PutEvents(*cloudwatchevents.PutEventsInput) (*cloudwatchevents.PutEventsOutput, error)
}

// EventStream is an implementation of the empire.EventStream interface backed
// by CloudWatch events..
type EventStream struct {
	// The topic to publish events to.
	Environment string

	client cloudwatcheventsClient
}

func NewEventStream(c client.ConfigProvider) *EventStream {
	return &EventStream{
		client: cloudwatchevents.New(c),
	}
}

func (e *EventStream) PublishEvent(event empire.Event) error {
	raw, err := json.Marshal(&Event{
		Environment: e.Environment,
		Event:       event.Event(),
		Message:     event.String(),
		Data:        event,
	})
	if err != nil {
		return err
	}

	entry := &cloudwatchevents.PutEventsRequestEntry{
		Detail:     aws.String(string(raw)),
		DetailType: aws.String(event.Event()),
		Source:     aws.String(Source),
	}

	_, err = e.client.PutEvents(&cloudwatchevents.PutEventsInput{
		Entries: []*cloudwatchevents.PutEventsRequestEntry{entry},
	})
	return err
}
