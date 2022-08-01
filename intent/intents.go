package intent

import (
	"github.com/discordpkg/gateway/event"
)

type Type int

const (
	Guilds Type = 1 << iota
	GuildMembers
	GuildBans
	GuildEmojisAndStickers
	GuildIntegrations
	GuildWebhooks
	GuildInvites
	GuildVoiceStates
	GuildPresences
	GuildMessages
	GuildMessageReactions
	GuildMessageTyping
	DirectMessages
	DirectMessageReactions
	DirectMessageTyping
	_
	GuildScheduledEvents
)

const Sum = DirectMessages | DirectMessageReactions | DirectMessageTyping | Guilds | GuildBans | GuildEmojisAndStickers | GuildIntegrations | GuildInvites | GuildMembers | GuildMessages | GuildMessageReactions | GuildMessageTyping | GuildPresences | GuildScheduledEvents | GuildVoiceStates | GuildWebhooks | 0

var intentsToEventsMap = map[Type][]event.Type{
	DirectMessages: {
		event.ChannelPinsUpdate,
		event.MessageCreate,
		event.MessageDelete,
		event.MessageUpdate,
	},
	DirectMessageReactions: {
		event.MessageReactionAdd,
		event.MessageReactionRemove,
		event.MessageReactionRemoveAll,
		event.MessageReactionRemoveEmoji,
	},
	DirectMessageTyping: {
		event.TypingStart,
	},
	Guilds: {
		event.ChannelCreate,
		event.ChannelDelete,
		event.ChannelPinsUpdate,
		event.ChannelUpdate,
		event.GuildCreate,
		event.GuildDelete,
		event.GuildRoleCreate,
		event.GuildRoleDelete,
		event.GuildRoleUpdate,
		event.GuildUpdate,
		event.StageInstanceCreate,
		event.StageInstanceDelete,
		event.StageInstanceUpdate,
		event.ThreadCreate,
		event.ThreadDelete,
		event.ThreadListSync,
		event.ThreadMembersUpdate,
		event.ThreadMemberUpdate,
		event.ThreadUpdate,
	},
	GuildBans: {
		event.GuildBanAdd,
		event.GuildBanRemove,
	},
	GuildEmojisAndStickers: {
		event.GuildEmojisUpdate,
		event.GuildStickersUpdate,
	},
	GuildIntegrations: {
		event.GuildIntegrationsUpdate,
		event.IntegrationCreate,
		event.IntegrationDelete,
		event.IntegrationUpdate,
	},
	GuildInvites: {
		event.InviteCreate,
		event.InviteDelete,
	},
	GuildMembers: {
		event.GuildCreate,
	},
	GuildMessages: {
		event.MessageCreate,
		event.MessageDelete,
		event.MessageDeleteBulk,
		event.MessageUpdate,
	},
	GuildMessageReactions: {
		event.MessageReactionAdd,
		event.MessageReactionRemove,
		event.MessageReactionRemoveAll,
		event.MessageReactionRemoveEmoji,
	},
	GuildMessageTyping: {
		event.TypingStart,
	},
	GuildPresences: {
		event.PresenceUpdate,
	},
	GuildScheduledEvents: {
		event.GuildScheduledEventCreate,
		event.GuildScheduledEventDelete,
		event.GuildScheduledEventUpdate,
		event.GuildScheduledEventUserAdd,
		event.GuildScheduledEventUserRemove,
	},
	GuildVoiceStates: {
		event.VoiceStateUpdate,
	},
	GuildWebhooks: {
		event.WebhooksUpdate,
	},
}

var emptyStruct struct{}
var dmIntents = map[Type]struct{}{
	DirectMessages:         emptyStruct,
	DirectMessageReactions: emptyStruct,
	DirectMessageTyping:    emptyStruct,
}

func Valid(intent Type) bool {
	return intent >= 0
}

func Events(intent Type) []event.Type {
	if events, ok := intentsToEventsMap[intent]; ok {
		cpy := make([]event.Type, len(events))
		copy(cpy, events)
		return cpy
	}
	return nil
}

func Merge(intents ...Type) Type {
	var merged Type
	for i := range intents {
		merged |= intents[i]
	}
	return merged
}

func DMEventsToIntents(src []event.Type) Type {
	return eventsToIntents(src, true)
}

func GuildEventsToIntents(src []event.Type) Type {
	return eventsToIntents(src, false)
}

func eventsToIntents(src []event.Type, dm bool) (intents Type) {
	contains := func(haystack []event.Type, needle event.Type) bool {
		for i := range haystack {
			if haystack[i] == needle {
				return true
			}
		}
		return false
	}

	for i := range src {
		for intent, events := range intentsToEventsMap {
			if _, isDM := dmIntents[intent]; (!dm && isDM) || (dm && !isDM) {
				continue
			}
			if contains(events, src[i]) {
				intents |= intent
			}
		}
	}

	return intents
}
