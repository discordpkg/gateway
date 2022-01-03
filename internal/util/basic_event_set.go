package util

import "github.com/andersfylling/discordgateway/event"

var emptyStruct = struct{}{}

type BasicEventSet map[event.Type]struct{}

func (s BasicEventSet) Add(events ...event.Type) {
	for i := range events {
		e := events[i]
		s[e] = emptyStruct
	}
}

func (s BasicEventSet) Remove(events ...event.Type) {
	for i := range events {
		e := events[i]
		delete(s, e)
	}
}

func (s BasicEventSet) Contains(evt event.Type) bool {
	_, ok := s[evt]
	return ok
}
