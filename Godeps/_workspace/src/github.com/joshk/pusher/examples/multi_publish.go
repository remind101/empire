package main

import (
	"fmt"
	"github.com/timonv/pusher"
	"time"
)

func main() {
	workers := 100
	messageCount := 5000
	messages := make(chan string)
	done := make(chan bool)

	client := pusher.NewClient("34420", "87bdfd3a6320e83b9289", "f25dfe88fb26ebf75139", false)

	for i := 0; i < workers; i++ {
		go func() {
			for data := range messages {
				err := client.Publish(data, "test", "test")
				if err != nil {
					fmt.Printf("E", err)
				} else {
					fmt.Print(".")
				}
			}
		}()
	}

	go func() {
		for i := 0; i < messageCount; i++ {
			messages <- "test"
		}
		done <- true
		close(messages)
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("\nDone :-)")
	case <-time.After(1 * time.Minute):
		fmt.Println("\nTimeout :-(")
	}

	fmt.Println("")
}
