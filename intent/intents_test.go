package intent

import (
	"testing"

	"github.com/andersfylling/discordgateway/event"
)

func TestAllAndSum(t *testing.T) {
	all := All()
	sum := Sum

	sumAll := Type(0)
	for i := range all {
		sumAll += all[i]
	}

	if sum != sumAll {
		t.Fatal(`"Sum" does not have the same value as sum of "All()"`)
	}
}

func TestDMEventsToIntents(t *testing.T) {
	type test struct {
		intent Type
		events []event.Type
	}

	table := []test{
		{DirectMessages, []event.Type{event.MessageCreate}},
		{DirectMessageTyping, []event.Type{event.GuildCreate, event.ChannelCreate, event.TypingStart}},
		{0, nil},
		{0, []event.Type{event.GuildCreate}},
	}

	for i := range table {
		derived := DMEventsToIntents(table[i].events)
		if derived != table[i].intent {
			t.Errorf("expected intent %d, got %d", table[i].intent, derived)
		}
	}
}
