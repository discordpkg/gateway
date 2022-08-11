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
<p>Use the existing disgord channels for discussion</p>
<p>
  <a href='https://discord.gg/fQgmBg'>
    <img src='https://img.shields.io/badge/Discord%20Gophers-%23disgord-blue.svg' alt='Discord Gophers' />
  </a>
  <a href='https://discord.gg/HBTHbme'>
    <img src='https://img.shields.io/badge/Discord%20API-%23disgord-blue.svg' alt='Discord API' />
  </a>
</p>

# Features

 - Complete control of goroutines (if desired)
 - Specify intents or GuildEvents & DirectMessageEvents
   - When events are used; intents are derived and redundant events pruned as soon as they are identified 
 - Receive Gateway events
 - Send Gateway commands
 - context support
 - Control over reconnect, disconnect, or behavior for handling discord errors


## Simple shard example 
> This code uses github.com/gobwas/ws, but you are free to use other
> websocket implementations as well. You just have to write your own Shard implementation
> and use GatewayState. See shard/shard.go for inspiration.

Here no handler is registered. Simply replace `nil` with a function pointer to read events (events with operation code 0).

Create a shard instance using the `gatewayshard` package:

```go
package main

import (
   "context"
   "errors"
   "fmt"
   "github.com/discordpkg/gateway"
   "github.com/discordpkg/gateway/event"
   "github.com/discordpkg/gateway/intent"
   "github.com/discordpkg/gateway/log"
   "github.com/discordpkg/gateway/shard"
   "net"
   "os"
)

func main() {
   shardInstance, err := shard.NewShard(0, os.Getenv("DISCORD_TOKEN"), nil,
      gateway.WithGuildEvents(event.All()...),
      gateway.WithDirectMessageEvents(intent.Events(intent.DirectMessageReactions)),
      gateway.WithIdentifyConnectionProperties(&discordgateway.IdentifyConnectionProperties{
         OS:      runtime.GOOS,
         Browser: "github.com/discordpkg/gateway v0",
         Device:  "tester",
      }),
   )
   if err != nil {
      log.Fatal(err)
   }

   dialUrl := "wss://gateway.discord.gg/?v=9&encoding=json"
```

You can then open a connection to discord and start listening for events. The event loop will continue to run
until the connection is lost or a process failed (json unmarshal/marshal, websocket frame issue, etc.)

You can use the helper methods for the DiscordError to decide when to reconnect:
```go
reconnectStage:
    if _, err := shardInstance.Dial(context.Background(), dialUrl); err != nil {
        log.Fatal("failed to open websocket connection. ", err)
    }

   if err = shardInstance.EventLoop(context.Background()); err != nil {
      reconnect := true

      var discordErr *gateway.DiscordError
      if errors.As(err, &discordErr) {
         reconnect = discordErr.CanReconnect()
      }

      if reconnect {
         logger.Infof("reconnecting: %s", discordErr.Error())
         if err := shardInstance.PrepareForReconnect(); err != nil {
            logger.Fatal("failed to prepare for reconnect:", err)
         }
         goto reconnectStage
      }
   }
}
```

Or manually check the close code, operation code, or error:
```go
reconnectStage:
   if _, err := shardInstance.Dial(context.Background(), dialUrl); err != nil {
      log.Fatal("failed to open websocket connection. ", err)
   }

   op, err := shardInstance.EventLoop(context.Background()); 
   if err != nil {
      var discordErr *gateway.DiscordError
      if errors.As(err, &discordErr) {
         switch discordErr.CloseCode {
         case 1001, 4000, 4007, 4009:
            // use reconnect logic defined later
         case 4001, 4002, 4003, 4004, 4005, 4008, 4010, 4011, 4012, 4013, 4014:
            log.Fatal("an error occured:", err)
         default:
            log.Fatal(fmt.Errorf("unhandled close error, with discord op code(%d): %d", op, discordErr.Code))
         }
      } else if !errors.Is(err, net.ErrClosed) {
         logger.Fatal("an error occured:", err)
      }
   }   
   
   if err := shardInstance.PrepareForReconnect(); err != nil {
      logger.Fatal("failed to prepare for reconnect:", err)
   }
   goto reconnectStage
}
```

## Gateway command
To request guild members, update voice state or update presence, you can utilize Shard.Write or GatewayState.Write (same logic). 
The bytes argument should not contain the discord payload wrapper (operation code, event name, etc.), instead you write only
the inner object and specify the relevant operation code.

> Calling Write(..) before dial or instantiating a net.Conn object will cause the process to fail. You must be connected.

```go

package main

import (
	"context"
	"fmt"
	"github.com/discordpkg/gateway"
	"github.com/discordpkg/gateway/intent"
	"github.com/discordpkg/gateway/command"
	"github.com/discordpkg/gateway/shard"
	"os"
)

func main() {
	shardInstance, err := shard.NewShard(0, os.Getenv("DISCORD_TOKEN"), nil,
		gateway.WithIntents(intent.Guilds),
	)
	if err != nil {
		panic(err)
	}

	dialUrl := "wss://gateway.discord.gg/?v=9&encoding=json"
	if _, err := shardInstance.Dial(context.Background(), dialUrl); err != nil {
       panic(fmt.Errorf("failed to open websocket connection. ", err))
	}

   // ...
   
	req := `{"guild_id":"23423","limit":0,"query":""}`
	if err := shardInstance.Write(command.RequestGuildMembers, []byte(req)); err != nil {
       panic(fmt.Errorf("failed to request guild members", err))
    }
    
}
```

If you need to manually set the intent value for whatever reason, the ShardConfig exposes an "Intents" field.
Note that intents will still be derived from DMEvents and GuildEvents and added to the final intents value used
to identify.

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
   - [ ] Commands
   - [x] JSON
   - [ ] ETF
   - [x] Rate limit
     - [x] Identify
     - [x] Commands
 - [ ] Shard(s) manager
 - [ ] Buffer pool
