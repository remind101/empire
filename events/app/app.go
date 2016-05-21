// Package app provides an empire.EventStream implementation that publishs
// events to the app's kinesis stream if available.
package app

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/remind101/empire"
)

type kinesisClient interface {
	PutRecord(*kinesis.PutRecordInput) (*kinesis.PutRecordOutput, error)
}

// EventStream is an implementation of the empire.EventStream interface backed
// by Kinesis.
type EventStream struct {
	empire  *empire.Empire
	kinesis kinesisClient
}

func NewEventStream(c client.ConfigProvider) *EventStream {
	return &EventStream{
		kinesis: kinesis.New(c),
	}
}

func (s *EventStream) PublishEvent(event empire.Event) error {
	if e, ok := event.(empire.AppEvent); ok {
		name := e.GetApp().ID
		key := fmt.Sprintf("%s.events", e.GetApp().ID)
		s.kinesis.PutRecord(&kinesis.PutRecordInput{
			Data:         []byte(e.String()),
			StreamName:   &name,
			PartitionKey: &key,
		})
	}
	return nil
}
