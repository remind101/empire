package main

import (
	"fmt"
	"github.com/timonv/pusher"
	"time"
)

func main() {
	client := pusher.NewClient("34420", "87bdfd3a6320e83b9289", "f25dfe88fb26ebf75139", false)

	done := make(chan bool)

	go func() {
		err := client.Publish("test", "test", "test")
		if err != nil {
			fmt.Printf("Error %s\n", err)
		} else {
			fmt.Println("Message Published!")
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
