package procs

import (
	"fmt"

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
		err = s.Repository.DeleteNode(n)
	}

	for addr, node := range sn {
		if _, ok := hn[addr]; !ok {
			delete(node)
		}
	}

	return err
}

// def run_once(self, **kwargs):
//     self.process_table.reap_minions()
//     try:
//         expected_procs = self.source.get_process_config() # Process counts we expect to be running
//     except MissingProcessConfig as e:
//         logger.warn("Missing process config at %s.", e.message)
//         logger.warn("Cowardly refusing to do anything.")
//         return
//     configured_processes = self.process_table.configured_processes() # Processes configured to run on minions
//     deleted_apps = set(configured_processes) - set(expected_procs)
//     for process_name, pd in expected_procs.items():
//         processes = configured_processes.get(process_name, [])
//         self.update_processes(process_name, pd, processes)
//         expected_count = pd.instance_count
//         configured_count = len(processes)
//         delta = expected_count - configured_count
//         if delta > 0:
//             self.scale_up(delta, pd)
//         if delta < 0:
//             self.scale_down(delta, pd)
//     if deleted_apps:
//         self.delete_apps(deleted_apps)

func (s *Scheduler) ScheduleProcs() error {
	unitsmap, err := s.UnitsService.FindAll() // Processes we expect to be running
	if err != nil {
		return err
	}

	procsmap, err := s.Repository.ProcessMap() // Processes currently scheduled
	if err != nil {
		return err
	}

	// Loop through expected processes, scaling accordingly
	for name, unit := range unitsmap {
		want := unit.InstanceCount
		got := len(procsmap[name])
		delta := want - got

		err := s.Scale(delta, unit)
		if err != nil {
			return err
		}
	}

	// Loop through scheduled procs, checking to see if they are still expected
	for name, procs := range procsmap {
		// Processes are no longer expected, should be removed from procsmap
		if _, ok := unitsmap[name]; !ok {
			for _, proc := range procs {
				s.DeleteProc(proc)
			}
		}
	}

	return nil
}

func (s *Scheduler) Scale(delta int, u units.Unit) error {
	switch {
	case delta == 0:
		return nil
	case delta < 0:
		for i := delta; i <= 0; i++ {
			if err := s.Schedule(u); err != nil {
				return err
			}
		}
	case delta > 0:
		for i := delta; i >= 0; i-- {
			if err := s.UnSchedule(u); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Scheduler) Schedule(u units.Unit) error {
	return nil
}

func (s *Scheduler) UnSchedule(u units.Unit) error {
	return nil
}

type Repository interface {
	DeleteNode(*Node) error
	DeleteProc(Process) error
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

func (c *consulRepository) DeleteNode(n *Node) error {
	_, err := c.client.KV().DeleteTree(NodesPrefix+n.Address, nil)
	return err
}

func (c *consulRepository) DeleteProc(p Process) error {
	_, err := c.client.KV().Delete(c.procKey(p), nil)
	return err
}

func (c *consulRepository) procKey(p Process) string {
	return fmt.Sprintf("%s/%s.%s.%d", p.Node.Address, p.AppID, p.ProcessType, p.ID)
}
