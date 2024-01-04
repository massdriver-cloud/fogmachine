package eventcache

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type Event struct {
	ResourceName       string
	ResourceStatus     string
	ProviderResourceID string
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

func (eventCache *EventCache) EventExists(eventID string) bool {
	_, ok := eventCache.Events[eventID]
	return ok
}

func (eventCache *EventCache) AddEvent(eventID string, event Event) {
	eventCache.Events[eventID] = event
}

func (eventCache EventCache) EventFromStack(event types.StackEvent, eventType string) Event {
	return Event{
		ResourceName:       *event.LogicalResourceId,
		ProviderResourceID: *event.PhysicalResourceId,
		ResourceStatus:     string(event.ResourceStatus),
		Type:               eventType,
	}
}
