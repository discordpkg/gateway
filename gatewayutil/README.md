# Gatewayutil

Utility package for gateway


## Simple shard example
> This code uses github.com/gobwas/ws, but you are free to use other
> websocket implementations as well. You just have to write your own Shard implementation
> and use gateway.Client. See gatewayutil/shard.go for inspiration.

Create a shard instance using the `gatewayutil` package:

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
   "github.com/discordpkg/gateway/gatewayutil"
   "net"
   "os"
)

func main() {
   shard, err := gatewayutil.NewShard(
      // gateway.WithLogger(&printLogger{}),
      gateway.WithBotToken(os.Getenv("DISCORD_TOKEN")),
      // gateway.WithEventHandler(someEventHandler),
      gateway.WithShardInfo(0, 1),
      gateway.WithGuildEvents(event.All()...),
      gateway.WithDirectMessageEvents(event.All()...),
      gateway.WithCommandRateLimiter(gatewayutil.NewCommandRateLimiter()),
      gateway.WithIdentifyRateLimiter(gatewayutil.NewLocalIdentifyRateLimiter()),
      gateway.WithIdentifyConnectionProperties(&gateway.IdentifyConnectionProperties{
         OS:      "linux",
         Browser: "github.com/discordpkg/gateway v0",
         Device:  "tester",
      }),
   )
   if err != nil {
      panic(err)
   }
```

You can then open a connection to discord and start listening for events. The event loop will continue to run
until the connection is lost or a process failed (json unmarshal/marshal, websocket frame issue, etc.)

You can use the helper methods for the DiscordError to decide when to reconnect:
```go
reconnectStage:
   _, err := shard.Dial(context.Background(), func() (string, error) {
      // code for calling GetGatewayBot url, only called if no resume url was cached from Discord 
	  return "wss://gateway.discord.gg/?v=10&encoding=json"
   })
   if err != nil {
      log.Fatal("failed to open websocket connection. ", err)
   }

   if err = shard.EventLoop(context.Background()); err != nil {
      var discordErr *gateway.DiscordError
      if errors.As(err, &discordErr) && discordErr.CanReconnect() {
         goto reconnectStage
      }
	  
	  return err
   }
}
```

Or manually check the close code, operation code, or error:
```go
   err := shard.EventLoop(context.Background()); 
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
   "github.com/discordpkg/gateway/event"
   "github.com/discordpkg/gateway/gatewayutil"
   "os"
)

func main() {
   shard, err := gatewayutil.NewShard(
      // gateway.WithLogger(&printLogger{}),
      gateway.WithBotToken(os.Getenv("DISCORD_TOKEN")),
      // gateway.WithEventHandler(someEventHandler),
      gateway.WithShardInfo(0, 1),
      gateway.WithGuildEvents(event.All()...),
      gateway.WithDirectMessageEvents(event.All()...),
      gateway.WithCommandRateLimiter(gatewayutil.NewCommandRateLimiter()),
      gateway.WithIdentifyRateLimiter(gatewayutil.NewLocalIdentifyRateLimiter()),
      gateway.WithIntents(intent.Guilds),
      gateway.WithCommandRateLimiter(gatewayutil.NewCommandRateLimiter()),
      gateway.WithIdentifyRateLimiter(gatewayutil.NewLocalIdentifyRateLimiter()),
   )
   if err != nil {
      panic(err)
   }

   if _, err := shard.Dial(context.Background(), client.GetGatewayBotURL); err != nil {
      panic(fmt.Errorf("failed to open websocket connection. ", err))
   }

   // ...

   req := `{"guild_id":"23423","limit":0,"query":""}`
   if err := shard.Write(event.RequestGuildMembers, []byte(req)); err != nil {
      panic(fmt.Errorf("failed to request guild members", err))
   }
}
```

If you need to manually set the intent value for whatever reason, the ShardConfig exposes an "Intents" field.
Note that intents will still be derived from DMEvents and GuildEvents and added to the final intents value used
to identify.
