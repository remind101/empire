package main

import (
	"fmt"
	"github.com/timonv/pusher"
	"time"
)

func main() {
	client := pusher.NewClient("4115", "23ed642e81512118260e", "cd72de5494540704dcf1", false)

	done := make(chan bool)

	go func() {
		queryParameters := map[string]string{
			"info":             "user_count",
			"filter_by_prefix": "presence-",
		}
		channels, err := client.Channels(queryParameters)
		if err != nil {
			fmt.Printf("Error %s\n", err)
		} else {
			fmt.Println(channels)
		}
		done <- true
	}()

	select {
	case <-done:
		fmt.Println("Done :-)")
	case <-time.After(1 * time.Minute):
		fmt.Println("Timeout :-(")
	}
}
