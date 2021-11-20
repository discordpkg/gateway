package event

// Code generated - This file has been automatically generated by _dev/main.go - DO NOT EDIT.

type Type string

const (
	// ChannelCreate
	ChannelCreate Type = "CHANNEL_CREATE"
	// ChannelDelete
	ChannelDelete Type = "CHANNEL_DELETE"
	// ChannelPinsUpdate
	ChannelPinsUpdate Type = "CHANNEL_PINS_UPDATE"
	// ChannelUpdate
	ChannelUpdate Type = "CHANNEL_UPDATE"
	// GuildBanAdd
	GuildBanAdd Type = "GUILD_BAN_ADD"
	// GuildBanRemove
	GuildBanRemove Type = "GUILD_BAN_REMOVE"
	// GuildCreate
	GuildCreate Type = "GUILD_CREATE"
	// GuildDelete
	GuildDelete Type = "GUILD_DELETE"
	// GuildEmojisUpdate
	GuildEmojisUpdate Type = "GUILD_EMOJIS_UPDATE"
	// GuildIntegrationsUpdate
	GuildIntegrationsUpdate Type = "GUILD_INTEGRATIONS_UPDATE"
	// GuildMembersChunk
	GuildMembersChunk Type = "GUILD_MEMBERS_CHUNK"
	// GuildMemberAdd
	GuildMemberAdd Type = "GUILD_MEMBER_ADD"
	// GuildMemberRemove
	GuildMemberRemove Type = "GUILD_MEMBER_REMOVE"
	// GuildMemberUpdate
	GuildMemberUpdate Type = "GUILD_MEMBER_UPDATE"
	// GuildRoleCreate
	GuildRoleCreate Type = "GUILD_ROLE_CREATE"
	// GuildRoleDelete
	GuildRoleDelete Type = "GUILD_ROLE_DELETE"
	// GuildRoleUpdate
	GuildRoleUpdate Type = "GUILD_ROLE_UPDATE"
	// GuildScheduledEventCreate
	GuildScheduledEventCreate Type = "GUILD_SCHEDULED_EVENT_CREATE"
	// GuildScheduledEventDelete
	GuildScheduledEventDelete Type = "GUILD_SCHEDULED_EVENT_DELETE"
	// GuildScheduledEventUpdate
	GuildScheduledEventUpdate Type = "GUILD_SCHEDULED_EVENT_UPDATE"
	// GuildScheduledEventUserAdd
	GuildScheduledEventUserAdd Type = "GUILD_SCHEDULED_EVENT_USER_ADD"
	// GuildScheduledEventUserRemove
	GuildScheduledEventUserRemove Type = "GUILD_SCHEDULED_EVENT_USER_REMOVE"
	// GuildStickersUpdate
	GuildStickersUpdate Type = "GUILD_STICKERS_UPDATE"
	// GuildUpdate
	GuildUpdate Type = "GUILD_UPDATE"
	// Hello
	Hello Type = "HELLO"
	// IntegrationCreate
	IntegrationCreate Type = "INTEGRATION_CREATE"
	// IntegrationDelete
	IntegrationDelete Type = "INTEGRATION_DELETE"
	// IntegrationUpdate
	IntegrationUpdate Type = "INTEGRATION_UPDATE"
	// InteractionCreate
	InteractionCreate Type = "INTERACTION_CREATE"
	// InvalidSession
	InvalidSession Type = "INVALID_SESSION"
	// InviteCreate
	InviteCreate Type = "INVITE_CREATE"
	// InviteDelete
	InviteDelete Type = "INVITE_DELETE"
	// MessageCreate
	MessageCreate Type = "MESSAGE_CREATE"
	// MessageDelete
	MessageDelete Type = "MESSAGE_DELETE"
	// MessageDeleteBulk
	MessageDeleteBulk Type = "MESSAGE_DELETE_BULK"
	// MessageReactionAdd
	MessageReactionAdd Type = "MESSAGE_REACTION_ADD"
	// MessageReactionRemove
	MessageReactionRemove Type = "MESSAGE_REACTION_REMOVE"
	// MessageReactionRemoveAll
	MessageReactionRemoveAll Type = "MESSAGE_REACTION_REMOVE_ALL"
	// MessageReactionRemoveEmoji
	MessageReactionRemoveEmoji Type = "MESSAGE_REACTION_REMOVE_EMOJI"
	// MessageUpdate
	MessageUpdate Type = "MESSAGE_UPDATE"
	// PresenceUpdate
	PresenceUpdate Type = "PRESENCE_UPDATE"
	// Ready
	Ready Type = "READY"
	// Reconnect
	Reconnect Type = "RECONNECT"
	// Resumed
	Resumed Type = "RESUMED"
	// StageInstanceCreate
	StageInstanceCreate Type = "STAGE_INSTANCE_CREATE"
	// StageInstanceDelete
	StageInstanceDelete Type = "STAGE_INSTANCE_DELETE"
	// StageInstanceUpdate
	StageInstanceUpdate Type = "STAGE_INSTANCE_UPDATE"
	// ThreadCreate
	ThreadCreate Type = "THREAD_CREATE"
	// ThreadDelete
	ThreadDelete Type = "THREAD_DELETE"
	// ThreadListSync
	ThreadListSync Type = "THREAD_LIST_SYNC"
	// ThreadMembersUpdate
	ThreadMembersUpdate Type = "THREAD_MEMBERS_UPDATE"
	// ThreadMemberUpdate
	ThreadMemberUpdate Type = "THREAD_MEMBER_UPDATE"
	// ThreadUpdate
	ThreadUpdate Type = "THREAD_UPDATE"
	// TypingStart
	TypingStart Type = "TYPING_START"
	// UserUpdate
	UserUpdate Type = "USER_UPDATE"
	// VoiceServerUpdate
	VoiceServerUpdate Type = "VOICE_SERVER_UPDATE"
	// VoiceStateUpdate
	VoiceStateUpdate Type = "VOICE_STATE_UPDATE"
	// WebhooksUpdate
	WebhooksUpdate Type = "WEBHOOKS_UPDATE"
)

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
