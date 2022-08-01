package event

// Code generated - This file has been automatically generated by _dev/main.go - DO NOT EDIT.

// Returns all 58 discord events
func All() []Type {
	return []Type{
		ChannelCreate,
		ChannelDelete,
		ChannelPinsUpdate,
		ChannelUpdate,
		GuildBanAdd,
		GuildBanRemove,
		GuildCreate,
		GuildDelete,
		GuildEmojisUpdate,
		GuildIntegrationsUpdate,
		GuildMembersChunk,
		GuildMemberAdd,
		GuildMemberRemove,
		GuildMemberUpdate,
		GuildRoleCreate,
		GuildRoleDelete,
		GuildRoleUpdate,
		GuildScheduledEventCreate,
		GuildScheduledEventDelete,
		GuildScheduledEventUpdate,
		GuildScheduledEventUserAdd,
		GuildScheduledEventUserRemove,
		GuildStickersUpdate,
		GuildUpdate,
		Hello,
		IntegrationCreate,
		IntegrationDelete,
		IntegrationUpdate,
		InteractionCreate,
		InvalidSession,
		InviteCreate,
		InviteDelete,
		MessageCreate,
		MessageDelete,
		MessageDeleteBulk,
		MessageReactionAdd,
		MessageReactionRemove,
		MessageReactionRemoveAll,
		MessageReactionRemoveEmoji,
		MessageUpdate,
		PresenceUpdate,
		Ready,
		Reconnect,
		Resumed,
		StageInstanceCreate,
		StageInstanceDelete,
		StageInstanceUpdate,
		ThreadCreate,
		ThreadDelete,
		ThreadListSync,
		ThreadMembersUpdate,
		ThreadMemberUpdate,
		ThreadUpdate,
		TypingStart,
		UserUpdate,
		VoiceServerUpdate,
		VoiceStateUpdate,
		WebhooksUpdate,
	}
}
