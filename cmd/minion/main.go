package main

import (
	"log"
	"os"

	"time"

	"github.com/hashicorp/consul/api"
	"github.com/remind101/empire/procs"
)

const (
	AgentTTL         = "30s"
	AgentServiceName = "empire-minions"
	AgentCheckID     = "service:empire-minions"
)

func main() {
	// Connect to consul
	client := newConsulClient()

	// Register agent
	register(client)

	// Heartbeat in a separate goroutine
	go heartBeat(client)

	// Check process config
	for _ = range time.NewTicker(time.Second * 10).C {
		checkConfig(client)
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

func register(c *api.Client) {
	check := &api.AgentServiceCheck{
		TTL: AgentTTL,
	}
	service := &api.AgentServiceRegistration{
		Name:  AgentServiceName,
		Check: check,
	}
	err := c.Agent().ServiceRegister(service)
	if err != nil {
		panic(err)
	}
}

func heartBeat(c *api.Client) {
	for _ = range time.NewTicker(time.Second * 5).C {
		err := c.Agent().PassTTL(AgentCheckID, "")
		if err != nil {
			log.Printf("Error in PassTTL(%s): %s\n", AgentCheckID, err)
		}
	}
}

func checkConfig(c *api.Client) {
	// Get list of processes scheduled for this agent,
	// register/deregister processes with init system.
	r := procs.NewConsulRepository(c)

	host, err := os.Hostname()
	if err != nil {
		log.Panicf("Unable to get hostname: %v\n", err)
	}
	m := &procs.Minion{Node: host}

	procs, err := r.MinionSchedule(m)
	if err != nil {
		log.Printf("Error getting schedule for Minion: %v\n", err)
	}

	for _, proc := range procs {
		log.Printf("Process scheduled for minion: %v\n", proc)
	}
}
