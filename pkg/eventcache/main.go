package eventcache

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type Event struct {
	ResourceName       string
	ResourceStatus     string
	ProviderResourceId string
	Message            string
	Type               string
}

type EventCache struct {
	Events map[string]Event
}

func New() *EventCache {
	return &EventCache{
		Events: make(map[string]Event),
	}
}

func (eventCache *EventCache) Prime(client *cloudformation.Client, stackName string) error {
	params := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	}

	result, err := client.DescribeStackEvents(context.Background(), params)

	if err != nil {
		return err
	}

	for _, event := range result.StackEvents {
		e := Event{
			ResourceName:       *event.LogicalResourceId,
			ProviderResourceId: *event.PhysicalResourceId,
			ResourceStatus:     string(event.ResourceStatus),
		}

		eventCache.Events[*event.EventId] = e
	}

	return nil
}

func (eventCache *EventCache) EventExists(eventId string) bool {
	_, ok := eventCache.Events[eventId]
	return ok
}

func (eventCache *EventCache) AddEvent(eventId string, event Event) {
	eventCache.Events[eventId] = event
}

func (eventCache EventCache) EventFromStack(event types.StackEvent, eventType string) Event {
	return Event{
		ResourceName:       *event.LogicalResourceId,
		ProviderResourceId: *event.PhysicalResourceId,
		ResourceStatus:     string(event.ResourceStatus),
		Type:               eventType,
	}
}
