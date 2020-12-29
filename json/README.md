Swap out the json implementation by overwriting the types and variables.

Here the standard json implementation is swapped out with jsoniter:
```go
package main

import (
    "github.com/andersfylling/discordgateway/json"
    jsoniter "github.com/json-iterator/go"
)

func init() {
    var j = jsoniter.ConfigCompatibleWithStandardLibrary
    
    json.Marshal = j.Marshal
    json.Unmarshal = j.Unmarshal	
}
```