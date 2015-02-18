package main

import (
	"log"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/remind101/empire/procs"
	"github.com/remind101/empire/units"
)

func main() {
	var stopCh <-chan struct{}

	// Connect to consul
	client := newConsulClient()

	scheduler := procs.NewScheduler(
		units.NewService(units.NewConsulRepository(client)),
		procs.NewConsulRepository(client),
	)

	lock, err := client.LockKey("/empire/scheduler/leader")
	if err != nil {
		panic(err)
	}

	for {
		// Block trying to aquire the lock
		lockCh, err := lock.Lock(stopCh)
		if err != nil {
			log.Printf("Error aquiring lock: %v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}

		ticker := time.NewTicker(time.Second * 10).C

		for {
			select {
			case <-ticker:
				err := scheduler.ReapMinions()
				if err != nil {
					log.Printf("Error in scheduler.ReapMinions: %s\n", err)
				}
				err = scheduler.ScheduleProcs()
				if err != nil {
					log.Printf("Error in scheduler.ScheduleProcs: %s\n", err)
				}
			case <-lockCh:
				// We lost the lock
				break
			}
		}
	}
}

func newConsulClient() *api.Client {
	conf := api.DefaultConfig()

	client, err := api.NewClient(conf)
	if err != nil {
		panic(err)
	}

	return client
}
