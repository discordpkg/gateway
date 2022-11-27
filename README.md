# Gateway
<p>
  <a href="https://codecov.io/gh/andersfylling/discordgateway">
    <img src="https://codecov.io/gh/andersfylling/discordgateway/branch/master/graph/badge.svg" />
  </a>
  <a href='https://goreportcard.com/report/github.com/discordpkg/gateway'>
    <img src='https://goreportcard.com/badge/github.com/discordpkg/gateway' alt='Code coverage' />
  </a>
  <a href='https://pkg.go.dev/github.com/discordpkg/gateway'>
    <img src="https://pkg.go.dev/badge/andersfylling/discordgateway" alt="PkgGoDev">
  </a>
</p>

A minimal implementation for the [Discord gateway](https://discord.com/developers/docs/topics/gateway) logic using 
the state pattern. Websocketing is separated out to allow re-use with any websocket library in golang. 
See [gatewayutil](./gatewayutil) for a shard implementation using [github.com/gobwas/ws](https://github.com/gobwas/ws).

# Features

 - Complete control of goroutines (if desired)
 - Intents
 - GuildEvents & DirectMessageEvents as a intent alternative for future-proofing / more optimizations 
 - Receive Gateway events
 - Send Gateway commands
 - context support
 - Control over reconnect, disconnect, or behavior for handling discord errors



## Identify rate limit
When you have multiple shards, you must inject a rate limiter for identify. The CommandRateLimitChan is optional in either case.
When no rate limiter for identifies are injected, one is created with the standard 1 identify per 5 second.

See the IdentifyRateLimiter interface for minimum implementation.

## Live bot for testing
There is a bot running the gobwas code. Found in the cmd subdir. If you want to help out the "stress testing", you can add the bot here: https://discord.com/oauth2/authorize?scope=bot&client_id=792491747711123486&permissions=0

It only reads incoming events and waits to crash. Once any alerts such as warning, error, fatal, panic triggers; I get a notification so I can quickly patch the problem!


## Support

 - [x] Gateway
   - [X] operation codes
   - [X] close codes
   - [X] Intents
   - [x] Events
   - [x] Commands
   - [x] JSON
   - [ ] ETF
   - [x] Rate limit
     - [x] Identify
     - [x] Commands
 - [ ] Shard(s) manager
 - [ ] Buffer pool


<p>Use the existing disgord channels for discussion</p>
<p>
  <a href='https://discord.gg/fQgmBg'>
    <img src='https://img.shields.io/badge/Discord%20Gophers-%23disgord-blue.svg' alt='Discord Gophers' />
  </a>
  <a href='https://discord.gg/HBTHbme'>
    <img src='https://img.shields.io/badge/Discord%20API-%23disgord-blue.svg' alt='Discord API' />
  </a>
</p>