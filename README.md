# Ideology

Discord is a mess. Consistency is a luxury. And simplicity is somewhere over there or here.

This project aims to normalize some namings, make interacting more intuitive and development a smoother experience.

Certain events and intents have been renamed in accordance to the famous CRUD naming scheme.

Philosophy/requirements:
 - Complete control of goroutines (if desired)
 - Events are bit flags and intents are a set of events (bit flags)
 - You're responsible for reading all incoming data
 - Sending gateway commands returns an error on failure
 - You only register for guild events by default (dm events must be stated explicitly)
 - context is supported when it makes sense, via ".WithContext(context.Context)"
 - desired events must be specified in the config, others are ignored (this allows for optimizations behind the scenes)
 - Control over reconnect, disconnect, or behavior for handling discord errors
 - cancellation

##


## Events and intents

> I don't agree with the way Discord allows subscribing to specific group of events or the previous 
"guild subscription" logic of theirs. Nor do I know what they will do in the future. And so I try to abstract this 
> away. You should only worry about the events you want, not intents, guild subscriptions, or whatever else might 
> be introduced later.

Intents are derived from events, except the special case of Direct Messaging capabilities. Those needs to be 
explicitly defined:

```go
intent.DirectMessages.EventFlags()
```

Events are defined as bit flags and intents are a bitmask consistent of a range of relevant events.

```go
event.GuildBanCreate = 0b0100000
event.GuildBanDelete = 0b1000000
--------------------------------
intent.GuildBans     = 0b1100000
```

You can specify a range of events to be ignored using bit operations:
```go
&ShardConfig{
    Events: ^(event.GuildBanCreate | event.GuildBanDelete),
}
```

Or just state you want whatever a certain intent holds:
```go
&ShardConfig{
    Events: intent.GuildMessages.EventFlags(),
}
```

Be aware that DM intents holds custom event flags. These are used internally to correctly understand which events you
are requesting.

Events not specified are discarded at the package level, and will not trigger the registered handler.

# Configuration

## Shard

### Intents

Intents are derived from event IDs. Intents that deals with DM events or events that can only take place when the shard ID is 0, must be provided explicitly.

Both intents and events are turned on and off by bit flags. Since intents states whether or not a specific range of events should be sent from Discord, we can say that a intent I subsumes a event E, if the bit value of E exists within the bit range of I.

intent.Guilds = 

Imagine that each event ID is a bit flag, and that intents 

```go
// I don't need direct message capabilities
conf := &discordgateway.ShardConfig{
    Events: event.MessageCreate | event.MessageDelete,
}
```

```go
// I need to handle direct messages
conf := &discordgateway.ShardConfig{
    Events: event.MessageCreate | event.MessageDelete,
    Intents: intent.DirectMessages, // explicitly stated
}
```

```go
// I need to handle direct messages
conf := &discordgateway.ShardConfig{
    Events: event.MessageCreate | event.MessageDelete,
    Intents: intent.DirectMessages | intent.GuildBans, // redundant intent, will error
}

// panic: intent.GuildBans does not subsume any given intent IDs 
```

## Support

 - [ ] Voice
 - [x] Gateway
   - [X] Intents
   - [x] JSON
   - [ ] ETF
   - [ ] Rate limit
     - [ ] Identify
     - [ ] Commands
 - [ ] Shard(s) manager
 - [ ] Buffer pool