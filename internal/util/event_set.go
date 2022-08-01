package util

import "github.com/discordpkg/gateway/event"

type EventSet interface {
	Add(events ...event.Type)
	Remove(events ...event.Type)
	Contains(evt event.Type) bool
}

func ToEventSlice(set EventSet) []event.Type {
	type EventsConverter interface {
		Events() []event.Type
	}
	if converter, ok := set.(EventsConverter); ok {
		return converter.Events()
	}

	switch t := set.(type) {
	case BasicEventSet:
		events := make([]event.Type, 0, len(t))
		for evt := range t {
			events = append(events, evt)
		}

		return events
	default:
		panic("unsupported type")
	}
}

func NewEventSet() EventSet {
	return make(BasicEventSet)
}
