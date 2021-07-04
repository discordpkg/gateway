## Flush & Write
Writing to a internal buffer to later call flush, is often seen for high throughput implementations. Given that discord rate limits any dispatched websocket message to 120/60s or ~2/1s, the throughput/write frequency is not high enough to merit a "write & flush" system. 
Instead, I want to deal with any errors as soon as possible. Immediately processing any dispatch errors as they happen, rather later at some point.

