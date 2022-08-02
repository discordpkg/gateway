package util

import (
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/event"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/opcode"
)

var emptyStruct = struct{}{}

type Set[T event.Type | intent.Type | opcode.Type | command.Type] map[T]struct{}

func (s Set[T]) Add(elements ...T) {
	for _, element := range elements {
		s[element] = emptyStruct
	}
}

func (s Set[T]) Remove(events ...T) {
	for _, element := range events {
		delete(s, element)
	}
}

func (s Set[T]) Contains(element T) bool {
	_, ok := s[element]
	return ok
}

func (s Set[T]) ToSlice() []T {
	elements := make([]T, 0, len(s))
	for element := range s {
		elements = append(elements, element)
	}

	return elements
}
