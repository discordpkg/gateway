package constants

// Ready contains the initial state information
const Ready = "READY"

// Resumed response to Resume
const Resumed = "RESUMED"

// ApplicationCommandCreate new Slash Command was created
const ApplicationCommandCreate = "APPLICATION_COMMAND_CREATE"

// ApplicationCommandUpdate Slash Command was updated
const ApplicationCommandUpdate = "APPLICATION_COMMAND_UPDATE"

// ApplicationCommandDelete Slash Command was deleted
const ApplicationCommandDelete = "APPLICATION_COMMAND_DELETE"

// ChannelCreate new guild channel created
const ChannelCreate = "CHANNEL_CREATE"

// ChannelUpdate channel was updated
const ChannelUpdate = "CHANNEL_UPDATE"

// ChannelDelete channel was deleted
const ChannelDelete = "CHANNEL_DELETE"

// ChannelPinsUpdate Sent when a message is pinned or unpinned in a text channel. This is not sent when a pinned message is deleted.
const ChannelPinsUpdate = "CHANNEL_PINS_UPDATE"

// ThreadCreate thread created, also sent when being added to a private thread
const ThreadCreate = "THREAD_CREATE"

// ThreadUpdate thread was updated
const ThreadUpdate = "THREAD_UPDATE"

// ThreadDelete thread was deleted
const ThreadDelete = "THREAD_DELETE"

// ThreadListSync sent when gaining access to a channel, contains all active threads in that channel
const ThreadListSync = "THREAD_LIST_SYNC"

// ThreadMemberUpdate thread member for the current user was updated
const ThreadMemberUpdate = "THREAD_MEMBER_UPDATE"

// ThreadMembersUpdate some user(s) were added to or removed from a thread
const ThreadMembersUpdate = "THREAD_MEMBERS_UPDATE"

// GuildCreate lazy-load for unavailable guild, guild became available, or user joined a new guild
const GuildCreate = "GUILD_CREATE"

// GuildUpdate guild was updated
const GuildUpdate = "GUILD_UPDATE"

// GuildDelete guild became unavailable, or user left/was removed from a guild
const GuildDelete = "GUILD_DELETE"

// GuildBanCreate user was banned from a guild
const GuildBanCreate = "GUILD_BAN_ADD"

// GuildBanDelete user was unbanned from a guild
const GuildBanDelete = "GUILD_BAN_REMOVE"

// GuildEmojisUpdate guild emojis were updated
const GuildEmojisUpdate = "GUILD_EMOJIS_UPDATE"

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

// IntegrationCreate guild integration was created
const IntegrationCreate = "INTEGRATION_CREATE"

// IntegrationUpdate guild integration was updated
const IntegrationUpdate = "INTEGRATION_UPDATE"

// IntegrationDelete guild integration was deleted
const IntegrationDelete = "INTEGRATION_DELETE"

// InteractionCreate user used an interaction, such as a Slash Command
const InteractionCreate = "INTERACTION_CREATE"

// InviteCreate	invite to a channel was created
const InviteCreate = "INVITE_CREATE"

// InviteDelete	invite to a channel was deleted
const InviteDelete = "INVITE_DELETE"

// MessageCreate message was created
const MessageCreate = "MESSAGE_CREATE"

// MessageUpdate message was edited
const MessageUpdate = "MESSAGE_UPDATE"

// MessageDelete message was deleted
const MessageDelete = "MESSAGE_DELETE"

// MessageDeleteBulk multiple messages were deleted at once
const MessageDeleteBulk = "MESSAGE_DELETE_BULK"

// MessageReactionCreate user reacted to a message
const MessageReactionCreate = "MESSAGE_REACTION_ADD"

// MessageReactionDelete user removed a reaction from a message
const MessageReactionDelete = "MESSAGE_REACTION_REMOVE"

// MessageReactionDeleteAll all reactions were explicitly removed from a message
const MessageReactionDeleteAll = "MESSAGE_REACTION_REMOVE_ALL"

// MessageReactionDeleteEmoji all reactions for a given emoji were explicitly removed from a message
const MessageReactionDeleteEmoji = "MESSAGE_REACTION_REMOVE_EMOJI"

// PresenceUpdate user was updated
const PresenceUpdate = "PRESENCE_UPDATE"

// TypingStart user started typing in a channel
const TypingStart = "TYPING_START"

// UserUpdate properties about the user changed
const UserUpdate = "USER_UPDATE"

// VoiceStateUpdate someone joined, left, or moved a voice channel
const VoiceStateUpdate = "VOICE_STATE_UPDATE"

// VoiceServerUpdate guild's voice server was updated
const VoiceServerUpdate = "VOICE_SERVER_UPDATE"

// WebhooksUpdate guild channel webhook was created, update, or deleted
const WebhooksUpdate = "WEBHOOKS_UPDATE"
