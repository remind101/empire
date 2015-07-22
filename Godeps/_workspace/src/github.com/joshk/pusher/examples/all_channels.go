package main

import (
	"fmt"
	"github.com/timonv/pusher"
	"sort"
	"time"
)

func main() {
	client := pusher.NewClient("4115", "23ed642e81512118260e", "cd72de5494540704dcf1", false)

	done := make(chan bool)

	go func() {
		channels, err := client.AllChannels()
		if err != nil {
			fmt.Printf("Error %s\n", err)
		} else {
			names := []string{}
			for k := range channels.List {
				names = append(names, k)
			}
			sort.Strings(names)
			fmt.Println("Channel Count:", len(names))
			fmt.Println(names)
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
