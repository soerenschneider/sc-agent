package events

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

type Event struct {
	Id     string
	Source string
	Type   string
	Data   string
}

var (
	ErrNoEventSinkConfigured = errors.New("no event sink is configured")
	ConfiguredEventSink      EventSink
	cloudEventSource         string
)

func init() {
	hostname, _ := os.Hostname()
	cloudEventSource = fmt.Sprintf("sc-agent://%s", hostname)
}

func Accept(ctx context.Context, event cloudevents.Event) error {
	if ConfiguredEventSink == nil {
		return ErrNoEventSinkConfigured
	}

	return ConfiguredEventSink.Accept(ctx, event)
}

func NewEvent(ctx context.Context, subject, eventType string, data any) error {
	event := cloudevents.NewEvent()
	event.SetSource(cloudEventSource)
	event.SetType(eventType)
	event.SetID(uuid.New().String())
	event.SetTime(time.Now())
	event.SetSubject(subject)
	if data != nil {
		event.SetDataContentType(cloudevents.ApplicationCloudEventsJSON)
		if err := event.SetData(cloudevents.ApplicationCloudEventsJSON, data); err != nil {
			return err
		}
	}

	return Accept(ctx, event)
}

type EventSink interface {
	Accept(ctx context.Context, event cloudevents.Event) error
}
