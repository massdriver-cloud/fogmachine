package eventcache

import (
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
