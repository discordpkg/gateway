<p>
  <a href="https://codecov.io/gh/andersfylling/discordgateway">
    <img src="https://codecov.io/gh/andersfylling/discordgateway/branch/master/graph/badge.svg" />
  </a>
  <a href='https://goreportcard.com/report/github.com/andersfylling/discordgateway'>
    <img src='https://goreportcard.com/badge/github.com/andersfylling/discordgateway' alt='Code coverage' />
  </a>
  <a href='https://pkg.go.dev/github.com/andersfylling/discordgateway'>
    <img src="https://pkg.go.dev/badge/andersfylling/discordgateway" alt="PkgGoDev">
  </a>
</p>
<p>Use the existing disgord channels for discussion</p>
<p>
  <a href='https://discord.gg/fQgmBg'>
    <img src='https://img.shields.io/badge/Discord%20Gophers-%23disgord-blue.svg' alt='Discord Gophers' />
  </a>
  <a href='https://discord.gg/HBTHbme'>
    <img src='https://img.shields.io/badge/Discord%20API-%23disgord-blue.svg' alt='Discord API' />
  </a>
</p>

# Ideology

Discord is a mess. Consistency is a luxury. And simplicity is somewhere over there or here.

This project aims to normalize some namings, make interacting more intuitive and development a smoother experience.

Certain events and intents have been renamed in accordance to the famous CRUD naming scheme.

Philosophy/requirements:
 - Complete control of goroutines (if desired)
 - intents are derived from GuildEvents and DMEvents specified in the configuration
- desired events must be specified in the config, others are ignored (this allows for optimizations behind the scenes)
 - You're responsible for reading all incoming data
 - Sending gateway commands returns an error on failure
 - context support
 - Control over reconnect, disconnect, or behavior for handling discord errors

## Simple shard example 
> This code uses github.com/gobwas/ws, but you are free to use other
> websocket implementations as well. You just have to write your own Shard implementation
> and use GatewayState. See shard.go for inspiration.

Here no handler is registered. Simply replace `nil` with a function pointer to read events (events with operation code 0). 
```go
shard, err := discordgateway.NewShard(nil, &discordgateway.ShardConfig{
    BotToken:            token,
    GuildEvents:         event.All(),
    DMEvents:            nil,
    TotalNumberOfShards: 1,
    IdentifyProperties: discordgateway.GatewayIdentifyProperties{
        OS:      "linux",
        Browser: "github.com/andersfylling/discordgateway v0",
        Device:  "tester",
    },
})
if err != nil {
    log.Fatal(err)
}

reconnect:
conn, err := shard.Dial(context.Background(), "wss://gateway.discord.gg/?v=8&encoding=json")
if err != nil {
    logger.Fatalf("failed to open websocket connection. %w", err)
}


if op, err := shard.EventLoop(context.Background(), conn); err != nil {
    var discordErr *discordgateway.CloseError
    if errors.As(err, &discordErr) {
        switch discordErr.Code {
        case 1001, 4000: // will initiate a resume
            fallthrough
        case 4007, 4009: // will do a fresh identify
            goto reconnect
        case 4001, 4002, 4003, 4004, 4005, 4008, 4010, 4011, 4012, 4013, 4014:
        default:
            logger.Errorf("unhandled close error, with discord op code(%d): %d", op, discordErr.Code)
        }
    }
    var errClosed *discordgateway.ErrClosed
    if errors.As(err, &errClosed) || errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe) {
        logger.Debug("connection closed/lost .. will try to reconnect")
        goto reconnect
    }
} else {
    goto reconnect
}
```

## Live bot for testing
There is a bot running the gobwas code. Found in the cmd subdir. If you want to help out the "stress testing", you can add the bot here: https://discord.com/oauth2/authorize?scope=bot&client_id=792491747711123486&permissions=0

It only reads incoming events and waits to crash. Once any alerts such as warning, error, fatal, panic triggers; I get a notification so I can quickly patch the problem!


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
