package intent

import (
	"testing"

	"github.com/andersfylling/discordgateway/event"
)

func TestGuildEventsToIntents(t *testing.T) {
	type test struct {
		intent Type
		events []event.Type
	}

	table := []test{
		{GuildMessages, []event.Type{event.MessageCreate}},
		{Guilds, []event.Type{event.GuildCreate, event.ChannelCreate}},
		{GuildBans | GuildEmojis | GuildIntegrations |
			GuildInvites | GuildMembers | GuildMessageReactions |
			GuildMessageTyping | GuildMessages | GuildPresences |
			GuildVoiceStates | GuildWebhooks | Guilds,
			event.All()},
	}

	for i := range table {
		derived := GuildEventsToIntents(table[i].events)
		if derived != table[i].intent {
			t.Errorf("expected intent %d, got %d", table[i].intent, derived)
		}
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

func TestAll(t *testing.T) {
	if All != 32767 {
		t.Fatalf("incorrect intent sum. Got %d, wants %d", All, 65534)
	}
}
