A Go Lang Pusher Library
========================

So much to write, so little information to tell you right now :)


## Example

```go
package main

import (
    "fmt"
    "github.com/timonv/pusher"
    "time"
)

func main() {
    client := pusher.NewClient("appId", "key", "secret", false)

    done := make(chan bool)

    go func() {
        err := client.Publish("test", "test", "test")
        if err != nil {
            fmt.Printf("Error %s\n", err)
        } else {
            fmt.Print("Message Published!")
        }
        done <- true
    }()

    // A basic timeout to make sure we don't wait forever
    select {
    case <-done:
        fmt.Println("\nDone")
    case <-time.After(1 * time.Minute):
        fmt.Println("\n:-( Timeout")
    }
}
```


## License

MIT: Timon Vonk and Josh Kalderimis http://timon-josh.mit-license.org