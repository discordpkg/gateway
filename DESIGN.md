## States
All websocket communication, read and writes, should flow through a state object (eg. gateway state) to ensure valid program behaviour. Once a connection terminates/closes for any reason, the state is also marked closed and becomes unusable. You must create a new state, even for resumes. This makes the lifetime of internal states easier to reason about, and no old configuration/behaviour is leaked into a new state.

## Flush & Write
Writing to a internal buffer to later call flush, is often seen for high throughput implementations. Given that discord rate limits any dispatched websocket message to 120/60s or ~2/1s, the throughput/write frequency is not high enough to merit a "write & flush" system. 
Instead, I want to deal with any errors as soon as possible. Immediately processing any dispatch errors as they happen, rather later at some point.

## Required communication and Rate limited commands
Commands can be split into high priority and low priority; high priority is any commands that keep the websocket connection alive (heartbeat, identify, resume, etc.), while low priority is user specified commands such as "request guild members".

The current system reserves around 5 command calls for every burst (120 commands/60s), for heartbeats. 
The system assumes that you've dispatched high priority commands to setup the connection, leaving only heartbeats to keep the connection alive.
However, it does mean that there will always be up to 5 wasted commands per burst. Implementing a dynamic high/low-priority queue would squeeze out any "lost" command calls, and is a welcomed PR.

## Dial & net.Conn
> Shard implementation is optional, but since it's part of the code base I'll still discuss it here.

The Dial method returns the net.Conn (ws connection) to allow you to create your own event loop system. It is not required to store the connection anywhere as the shard keeps a reference when Dial succeeds.

## Error handling
Just like the go std packages, the error syntax is:
 - Err* for variables
 - *Error for struct implementations

FrameError is wraps any internal error that happens when processing a frame. Any frame errors happens before the payload content can be read by the state. Try reconnecting.

## Rate Limiting Identify
Given that identify may have to be shared across shards, the rate limiter must be inject-able.

This extends to microservice support; I've kept the interface as a simple "take" design. The method takes a shard id and returns a bool to signify if the shard can identify. 
The return value does not seem to useful as of now, please open a issue or PR with any ideas/feedback you may have.