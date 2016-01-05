package tugboat

import (
	"errors"
	"net/url"
	"strings"
)

// Pusher represents a pusher client that can publish events to a channel.
type Pusher interface {
	Publish(data, event string, channels ...string) error
}

// nullPusher is a pusher client that does nothing.
type nullPusher struct{}

func (p *nullPusher) Publish(data, event string, channels ...string) error {
	return nil
}

type event struct {
	data     string
	event    string
	channels []string
}

// asyncPusher is a pusher client that sends pusher events in a go routine.
type asyncPusher struct {
	pusher Pusher
	events chan *event
}

func newAsyncPusher(pusher Pusher, enqueue int) *asyncPusher {
	p := &asyncPusher{
		pusher: pusher,
		events: make(chan *event, enqueue),
	}
	go p.start()
	return p
}

func (p *asyncPusher) Publish(data, evt string, channels ...string) error {
	p.events <- &event{
		data:     data,
		event:    evt,
		channels: channels,
	}
	return nil
}

func (p *asyncPusher) start() {
	for event := range p.events {
		p.pusher.Publish(event.data, event.event, event.channels...)
	}
}

type PusherCredentials struct {
	AppID  string
	Key    string
	Secret string
}

func ParsePusherCredentials(uri string) (*PusherCredentials, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	if u.User == nil {
		return nil, errors.New("no credentials provided for pusher")
	}

	key := u.User.Username()
	secret, ok := u.User.Password()
	if !ok {
		return nil, errors.New("no password provided")
	}

	appID := strings.Replace(u.Path, "/apps/", "", 1)

	return &PusherCredentials{
		AppID:  appID,
		Key:    key,
		Secret: secret,
	}, nil
}
