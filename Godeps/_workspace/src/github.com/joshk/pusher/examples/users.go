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
		users, err := client.Users("common")
		if err != nil {
			fmt.Printf("Error %s\n", err)
		} else {
			ids := []int{}
			for k := range users.List {
				ids = append(ids, k)
			}
			sort.Ints(ids)
			fmt.Println("User Count:", len(ids))
			fmt.Println(ids)
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
