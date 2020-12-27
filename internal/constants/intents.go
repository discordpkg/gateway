package constants

import "github.com/andersfylling/discordgateway/event"

type Intent uint64

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
	Events event.Flag
	Intent Intent
}

var Guilds = relation{
	Events: event.GuildCreate | event.GuildUpdate | event.GuildDelete |
		event.GuildRoleCreate | event.GuildRoleUpdate | event.GuildRoleDelete |
		event.ChannelCreate | event.ChannelUpdate | event.ChannelDelete |
		event.ChannelPinsUpdate,
	Intent: guilds,
}

var GuildMembers = relation{
	Events: event.GuildMemberCreate | event.GuildMemberUpdate | event.GuildMemberDelete,
	Intent: guildMembers,
}

var GuildBans = relation{
	Events: event.GuildBanCreate | event.GuildBanDelete,
	Intent: guildBans,
}

var GuildEmojis = relation{
	Events: event.GuildEmojisUpdate,
	Intent: guildEmojis,
}

var GuildIntegrations = relation{
	Events: event.GuildIntegrationsUpdate,
	Intent: guildIntegrations,
}

var GuildWebhooks = relation{
	Events: event.WebhooksUpdate,
	Intent: guildWebhooks,
}

var GuildInvites = relation{
	Events: event.InviteCreate | event.InviteDelete,
	Intent: guildInvites,
}

var GuildVoiceStates = relation{
	Events: event.VoiceStateUpdate,
	Intent: guildVoiceStates,
}

var GuildPresences = relation{
	Events: event.PresenceUpdate,
	Intent: guildPresences,
}

var GuildMessages = relation{
	Events: event.MessageCreate | event.MessageUpdate | event.MessageDelete |
		event.MessageDeleteBulk,
	Intent: guildMessages,
}

var GuildMessageReactions = relation{
	Events: event.MessageReactionCreate | event.MessageReactionDelete | event.MessageReactionDeleteAll |
		event.MessageReactionDeleteEmoji,
	Intent: guildMessageReactions,
}

var GuildMessageTyping = relation{
	Events: event.TypingStart,
	Intent: guildMessageTyping,
}

var DirectMessages = relation{
	Events: event.ChannelCreate | event.MessageCreate | event.MessageUpdate |
		event.MessageDelete | event.ChannelPinsUpdate,
	Intent: directMessages,
}

var DirectMessageReactions = relation{
	Events: event.MessageReactionCreate | event.MessageReactionDelete | event.MessageReactionDeleteAll |
		event.MessageReactionDeleteEmoji,
	Intent: directMessageReactions,
}

var DirectMessageTyping = relation{
	Events: event.TypingStart,
	Intent: directMessageTyping,
}
