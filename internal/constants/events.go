package constants

// Ready The ready event is dispatched when a client has completed the initial handshake with the gateway (for new sessions).
// The ready event can be the largest and most complex event the gateway will send, as it contains all the state
// required for a client to begin interacting with the rest of the platform.
const Ready = "READY"

// Resumed The resumed event is dispatched when a client has sent a resume payload to the gateway
// (for resuming existing sessions).
const Resumed = "RESUMED"

// ChannelCreate Sent when a new channel is created, relevant to the current user. The inner payload is a DM channel or
// guild channel object.
const ChannelCreate = "CHANNEL_CREATE"

// ChannelUpdate Sent when a channel is updated. The inner payload is a guild channel object.
const ChannelUpdate = "CHANNEL_UPDATE"

// ChannelDelete Sent when a channel relevant to the current user is deleted. The inner payload is a DM or Guild channel object.
const ChannelDelete = "CHANNEL_DELETE"

// ChannelPinsUpdate Sent when a message is pinned or unpinned in a text channel. This is not sent when a pinned message is deleted.
const ChannelPinsUpdate = "CHANNEL_PINS_UPDATE"

// TypingStart Sent when a user starts typing in a channel.
const TypingStart = "TYPING_START"

// InviteDelete Sent when an invite is deleted.
const InviteDelete = "INVITE_DELETE"

// MessageCreate Sent when a message is created. The inner payload is a message object.
const MessageCreate = "MESSAGE_CREATE"

// MessageUpdate Sent when a message is updated. The inner payload is a message object.
//
// NOTE! Has _at_least_ the GuildID and ChannelID fields.
const MessageUpdate = "MESSAGE_UPDATE"

// MessageDelete Sent when a message is deleted.
const MessageDelete = "MESSAGE_DELETE"

// MessageDeleteBulk Sent when multiple messages are deleted at once.
const MessageDeleteBulk = "MESSAGE_DELETE_BULK"

// MessageReactionCreate Sent when a user adds a reaction to a message.
const MessageReactionCreate = "MESSAGE_REACTION_ADD"

// MessageReactionDelete Sent when a user removes a reaction from a message.
const MessageReactionDelete = "MESSAGE_REACTION_REMOVE"

// MessageReactionDeleteAll Sent when a user explicitly removes all reactions from a message.
const MessageReactionDeleteAll = "MESSAGE_REACTION_REMOVE_ALL"

// GuildEmojisUpdate Sent when a guild's emojis have been updated.
const GuildEmojisUpdate = "GUILD_EMOJIS_UPDATE"

// GuildCreate This event can be sent in three different scenarios:
//  1. When a user is initially connecting, to lazily load and backfill information for all unavailable guilds
//     sent in the Ready event.
//	2. When a Guild becomes available again to the client.
// 	3. When the current user joins a new Guild.
const GuildCreate = "GUILD_CREATE"

// GuildUpdate Sent when a guild is updated. The inner payload is a guild object.
const GuildUpdate = "GUILD_UPDATE"

// GuildDelete Sent when a guild becomes unavailable during a guild outage, or when the user leaves or is removed from a guild.
// The inner payload is an unavailable guild object. If the unavailable field is not set, the user was removed
// from the guild.
const GuildDelete = "GUILD_DELETE"

// GuildBanCreate Sent when a user is banned from a guild. The inner payload is a user object, with an extra guild_id key.
const GuildBanCreate = "GUILD_BAN_ADD"

// GuildBanDelete Sent when a user is unbanned from a guild. The inner payload is a user object, with an extra guild_id key.
const GuildBanDelete = "GUILD_BAN_REMOVE"

// GuildIntegrationsUpdate Sent when a guild integration is updated.
const GuildIntegrationsUpdate = "GUILD_INTEGRATIONS_UPDATE"

// GuildMemberCreate Sent when a new user joins a guild. The inner payload is a guild member object with these extra fields:
const GuildMemberCreate = "GUILD_MEMBER_ADD"

// GuildMemberDelete Sent when a user is removed from a guild (leave/kick/ban).
const GuildMemberDelete = "GUILD_MEMBER_REMOVE"

// GuildMemberUpdate Sent when a guild member is updated.
const GuildMemberUpdate = "GUILD_MEMBER_UPDATE"

// GuildMembersChunk Sent in response to Gateway Request Guild Members.
const GuildMembersChunk = "GUILD_MEMBERS_CHUNK"

// GuildRoleCreate Sent when a guild role is created.
const GuildRoleCreate = "GUILD_ROLE_CREATE"

// GuildRoleUpdate Sent when a guild role is created.
const GuildRoleUpdate = "GUILD_ROLE_UPDATE"

// GuildRoleDelete Sent when a guild role is created.
const GuildRoleDelete = "GUILD_ROLE_DELETE"

// PresenceUpdate A user's presence is their current state on a guild. This event is sent when a user's presence is updated for a guild.
const PresenceUpdate = "PRESENCE_UPDATE"

// UserUpdate Sent when properties about the user change. Inner payload is a user object.
const UserUpdate = "USER_UPDATE"

// VoiceStateUpdate Sent when someone joins/leaves/moves voice channels. Inner payload is a voice state object.
const VoiceStateUpdate = "VOICE_STATE_UPDATE"

// VoiceServerUpdate Sent when a guild's voice server is updated. This is sent when initially connecting to voice, and when the current
// voice instance fails over to a new server.
const VoiceServerUpdate = "VOICE_SERVER_UPDATE"

// WebhooksUpdate Sent when a guild channel's WebHook is created, updated, or deleted.
const WebhooksUpdate = "WEBHOOKS_UPDATE"

// InviteCreate Sent when a guild's invite is created.
const InviteCreate = "INVITE_CREATE"

// MessageReactionDeleteEmoji Sent when a bot removes all instances of a given emoji from the reactions of a message.
const MessageReactionDeleteEmoji = "MESSAGE_REACTION_REMOVE_EMOJI"

// InteractionCreate Sent when a user in a guild uses a Slash Command. Inner payload is an Interaction.
const InteractionCreate = "INTERACTION_CREATE"
