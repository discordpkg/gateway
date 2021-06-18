package constants

import "github.com/andersfylling/discordgateway/event"

type Intent uint32

const (
	GuildsVal Intent = 1 << iota
	GuildMembersVal
	GuildBansVal
	GuildEmojisVal
	GuildIntegrationsVal
	GuildWebhooksVal
	GuildInvitesVal
	GuildVoiceStatesVal
	GuildPresencesVal
	GuildMessagesVal
	GuildMessageReactionsVal
	GuildMessageTypingVal

	DirectMessagesVal
	DirectMessageReactionsVal
	DirectMessageTypingVal
)

type relation struct {
	Events []event.Type
	Intent Intent
}

var Guilds = relation{
	Events: []event.Type{
		event.GuildCreate, event.GuildUpdate, event.GuildDelete,
		event.GuildRoleCreate, event.GuildRoleUpdate, event.GuildRoleDelete,
		event.ChannelCreate, event.ChannelUpdate, event.ChannelDelete, event.ChannelPinsUpdate,
		event.ThreadCreate, event.ThreadUpdate, event.ThreadDelete, event.ThreadListSync,
		event.ThreadMemberUpdate, event.ThreadMembersUpdate,
	},
	Intent: GuildsVal,
}

var GuildMembers = relation{
	Events: []event.Type{
		event.GuildMemberCreate, event.GuildMemberUpdate, event.GuildMemberDelete,
		event.ThreadMembersUpdate,
	},
	Intent: GuildMembersVal,
}

var GuildBans = relation{
	Events: []event.Type{event.GuildBanCreate, event.GuildBanDelete},
	Intent: GuildBansVal,
}

var GuildEmojis = relation{
	Events: []event.Type{event.GuildEmojisUpdate},
	Intent: GuildEmojisVal,
}

var GuildIntegrations = relation{
	Events: []event.Type{
		event.GuildIntegrationsUpdate,
		event.IntegrationCreate, event.IntegrationUpdate, event.IntegrationDelete,
	},
	Intent: GuildIntegrationsVal,
}

var GuildWebhooks = relation{
	Events: []event.Type{event.WebhooksUpdate},
	Intent: GuildWebhooksVal,
}

var GuildInvites = relation{
	Events: []event.Type{event.InviteCreate, event.InviteDelete},
	Intent: GuildInvitesVal,
}

var GuildVoiceStates = relation{
	Events: []event.Type{event.VoiceStateUpdate},
	Intent: GuildVoiceStatesVal,
}

var GuildPresences = relation{
	Events: []event.Type{event.PresenceUpdate},
	Intent: GuildPresencesVal,
}

var GuildMessages = relation{
	Events: []event.Type{
		event.MessageCreate, event.MessageUpdate, event.MessageDelete,
		event.MessageDeleteBulk,
	},
	Intent: GuildMessagesVal,
}

var GuildMessageReactions = relation{
	Events: []event.Type{
		event.MessageReactionCreate, event.MessageReactionDelete, event.MessageReactionDeleteAll,
		event.MessageReactionDeleteEmoji,
	},
	Intent: GuildMessageReactionsVal,
}

var GuildMessageTyping = relation{
	Events: []event.Type{event.TypingStart},
	Intent: GuildMessageTypingVal,
}

var DirectMessages = relation{
	Events: []event.Type{
		event.ChannelCreate, event.MessageCreate, event.MessageUpdate,
		event.MessageDelete, event.ChannelPinsUpdate,
	},
	Intent: DirectMessagesVal,
}

var DirectMessageReactions = relation{
	Events: []event.Type{
		event.MessageReactionCreate, event.MessageReactionDelete, event.MessageReactionDeleteAll,
		event.MessageReactionDeleteEmoji,
	},
	Intent: DirectMessageReactionsVal,
}

var DirectMessageTyping = relation{
	Events: []event.Type{event.TypingStart},
	Intent: DirectMessageTypingVal,
}
