package procs

import (
	"github.com/hashicorp/consul/api"
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
	"github.com/remind101/empire/units"
)

const (
	NodesServiceName = "empire-nodes"
	NodesPrefix      = "/empire/nodes/"
)

type Address string
type Node api.Node

// Process represents a process that has been scheduled to run
// on a minion. Processes are scheduled based on the Units map.
//
// TODO Units is a poor name, rename to something like Process
// Blueprint or Plan or something.

type Process struct {
	ID          int
	AppID       apps.ID
	ReleaseID   releases.ID
	ProcessType slugs.ProcessType
	Node        *Node
}

type Processes []Process

type ProcessMap map[units.Name]Processes

type NodeMap map[Address]*Node

type Scheduler struct {
	UnitsService *units.Service // Service to fetch the desired process state from
	Repository                  // Repository for scheduling processes to minions
}

func (s *Scheduler) ReapNodes() error {
	var err error
	var sn, hn NodeMap // scheduled, healthy

	if sn, err = s.Repository.ScheduledNodes(); err != nil {
		return err
	}

	if hn, err = s.Repository.HealthyNodes(); err != nil {
		return err
	}

	delete := func(n *Node) {
		if err != nil {
			return
		}
		err = s.Repository.Delete(n)
	}

	for addr, node := range sn {
		if _, ok := hn[addr]; !ok {
			delete(node)
		}
	}

	return err
}

// func (s *Scheduler) ScheduleUnits() error {
// 	unitsmap, err := s.UnitsService.FindAll()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

type Repository interface {
	Delete(*Node) error
	ProcessMap() (ProcessMap, error)  // The current ProcessMap
	HealthyNodes() (NodeMap, error)   // Nodes that are reporting themselves as healthy
	ScheduledNodes() (NodeMap, error) // Nodes that are actively scheduled.
}

type consulRepository struct {
	client *api.Client
}

func (c *consulRepository) HealthyNodes() (NodeMap, error) {
	entries, _, err := c.client.Health().Service(NodesServiceName, "", true, nil)
	if err != nil {
		return NodeMap{}, err
	}

	nodemap := make(NodeMap, len(entries))

	for _, entry := range entries {
		n := Node(*entry.Node)
		nodemap[Address(entry.Node.Address)] = &n
	}

	return nodemap, err
}

func (c *consulRepository) ScheduledNodes() (NodeMap, error) {
	addrs, _, err := c.client.KV().Keys(NodesPrefix, "/", nil)
	if err != nil {
		return NodeMap{}, err
	}

	nodemap := make(NodeMap, len(addrs))

	for _, addr := range addrs {
		nodemap[Address(addr)] = &Node{Address: addr}
	}

	return nodemap, err
}

func (c *consulRepository) Delete(n *Node) error {
	_, err := c.client.KV().DeleteTree(NodesPrefix+n.Address, nil)
	return err
}
