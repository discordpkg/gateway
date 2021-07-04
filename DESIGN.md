## Flush & Write
Writing to a internal buffer to later call flush, is often seen for high throughput implementations. Given that discord rate limits any dispatched websocket message to 120/60s or ~2/1s, the throughput/write frequency is not high enough to merit a "write & flush" system. 
Instead, I want to deal with any errors as soon as possible. Immediately processing any dispatch errors as they happen, rather later at some point.

## Required communication and Rate limited commands
Commands can be split into high priority and low priority; high priority is any commands that keep the websocket connection alive (heartbeat, identify, resume, etc.), while low priority is user specified commands such as "request guild members".

The current system reserves around 5 command calls for every burst (120 commands/60s), for heartbeats. 
The system assumes that you've dispatched high priority commands to setup the connection, leaving only heartbeats to keep the connection alive.
However, it does mean that there will always be up to 5 wasted commands per burst. Implementing a dynamic high/low-priority queue would squeeze out any "lost" command calls, and is a welcomed PR.