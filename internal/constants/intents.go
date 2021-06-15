package constants

import "github.com/andersfylling/discordgateway/event"

type Intent uint32

const (
	guilds Intent = 0b1 << iota
	guildMembers
	guildBans
	guildEmojis
	guildIntegrations
	guildWebhooks
	guildInvites
	guildVoiceStates
	guildPresences
	guildMessages
	guildMessageReactions
	guildMessageTyping

	directMessages
	directMessageReactions
	directMessageTyping
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
	Intent: guilds,
}

var GuildMembers = relation{
	Events: []event.Type{
		event.GuildMemberCreate, event.GuildMemberUpdate, event.GuildMemberDelete,
		event.ThreadMembersUpdate,
	},
	Intent: guildMembers,
}

var GuildBans = relation{
	Events: []event.Type{event.GuildBanCreate, event.GuildBanDelete},
	Intent: guildBans,
}

var GuildEmojis = relation{
	Events: []event.Type{event.GuildEmojisUpdate},
	Intent: guildEmojis,
}

var GuildIntegrations = relation{
	Events: []event.Type{
		event.GuildIntegrationsUpdate,
		event.IntegrationCreate, event.IntegrationUpdate, event.IntegrationDelete,
	},
	Intent: guildIntegrations,
}

var GuildWebhooks = relation{
	Events: []event.Type{event.WebhooksUpdate},
	Intent: guildWebhooks,
}

var GuildInvites = relation{
	Events: []event.Type{event.InviteCreate, event.InviteDelete},
	Intent: guildInvites,
}

var GuildVoiceStates = relation{
	Events: []event.Type{event.VoiceStateUpdate},
	Intent: guildVoiceStates,
}

var GuildPresences = relation{
	Events: []event.Type{event.PresenceUpdate},
	Intent: guildPresences,
}

var GuildMessages = relation{
	Events: []event.Type{
		event.MessageCreate, event.MessageUpdate, event.MessageDelete,
		event.MessageDeleteBulk,
	},
	Intent: guildMessages,
}

var GuildMessageReactions = relation{
	Events: []event.Type{
		event.MessageReactionCreate, event.MessageReactionDelete, event.MessageReactionDeleteAll,
		event.MessageReactionDeleteEmoji,
	},
	Intent: guildMessageReactions,
}

var GuildMessageTyping = relation{
	Events: []event.Type{event.TypingStart},
	Intent: guildMessageTyping,
}

var DirectMessages = relation{
	Events: []event.Type{
		event.ChannelCreate, event.MessageCreate, event.MessageUpdate,
		event.MessageDelete, event.ChannelPinsUpdate,
	},
	Intent: directMessages,
}

var DirectMessageReactions = relation{
	Events: []event.Type{
		event.MessageReactionCreate, event.MessageReactionDelete, event.MessageReactionDeleteAll,
		event.MessageReactionDeleteEmoji,
	},
	Intent: directMessageReactions,
}

var DirectMessageTyping = relation{
	Events: []event.Type{event.TypingStart},
	Intent: directMessageTyping,
}
