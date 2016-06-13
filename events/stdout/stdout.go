// Package stdout provides an empire.EventStream implementation that publishes
// events to stdout.
package stdout

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/remind101/empire"
)

type EventStream struct{}

func NewEventStream(c client.ConfigProvider) *EventStream {
	return &EventStream{}
}

func (e *EventStream) PublishEvent(event empire.Event) error {
	_, err := fmt.Println(event.String())
	return err
}
