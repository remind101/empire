package main

import (
	"log"

	"time"

	"github.com/hashicorp/consul/api"
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
	for range time.NewTicker(time.Second * 10).C {
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
	for range time.NewTicker(time.Second * 5).C {
		err := c.Agent().PassTTL(AgentCheckID, "")
		if err != nil {
			log.Printf("Error in PassTTL(%s): %s", AgentCheckID, err)
		}
	}
}

func checkConfig(c *api.Client) {
	// Get list of processes scheduled for this agent,
	// register/deregister processes with init system.
}