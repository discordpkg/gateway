Swap out the json implementation by overwriting the types and variables.

Here the standard json implementation is swapped out with jsoniter:
```go
package main

import (
    "github.com/discordpkg/gateway/encoding"
    jsoniter "github.com/json-iterator/go"
)

func init() {
    var j = jsoniter.ConfigCompatibleWithStandardLibrary
    
    encoding.Marshal = j.Marshal
	encoding.Unmarshal = j.Unmarshal	
}
```

The idea here is that you may also switch to a different format such as [ETF](https://discord.com/developers/docs/topics/gateway#encoding-and-compression):
```go
package main

import (
	"github.com/JakeMakesStuff/go-erlpack"
	"github.com/discordpkg/gateway/encoding"
)

func init() {
	encoding.Marshal = erlpack.Pack
	encoding.Unmarshal = erlpack.Unpack
}
```
