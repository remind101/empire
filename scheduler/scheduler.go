package scheduler

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
	"github.com/remind101/empire/units"
)

const (
	MinionsServiceName = "empire-minions"
	MinionsPrefix      = "/empire/minions/"
)

type Node string // api.Node.Node, a.k.a the hostname of the node
type Minion api.Node

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
	Minion      *Minion
}

func (p *Process) Name() units.Name {
	return units.GenName(p.AppID, p.ProcessType)
}

type Processes []Process

type ProcessMap map[units.Name]Processes

type MinionMap map[Node]*Minion

type Scheduler struct {
	UnitsService *units.Service // Service to fetch the desired process state from
	Repository                  // Repository for scheduling processes to minions
}

func New(u *units.Service, r Repository) *Scheduler {
	return &Scheduler{UnitsService: u, Repository: r}
}

func (s *Scheduler) ReapMinions() error {
	var err error
	var sm, hm MinionMap // scheduled, healthy

	if sm, err = s.Repository.ScheduledMinions(); err != nil {
		return err
	}

	if hm, err = s.Repository.HealthyMinions(); err != nil {
		return err
	}

	delete := func(m *Minion) {
		if err != nil {
			return
		}
		err = s.Repository.DeleteMinion(m)
	}

	for node, minion := range sm {
		if _, ok := hm[node]; !ok {
			delete(minion)
		}
	}

	return err
}

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
	DeleteMinion(*Minion) error
	DeleteProc(Process) error
	ProcessMap() (ProcessMap, error)           // The current ProcessMap
	HealthyMinions() (MinionMap, error)        // Minions that are reporting themselves as healthy
	ScheduledMinions() (MinionMap, error)      // Minions that are actively scheduled.
	MinionSchedule(*Minion) (Processes, error) // Processes scheduled for a particular Minion
}

// consulRepository is an implementation of Repository backed by Consul.
//
// It uses the following key pattern to store the process map:
//
//     /empire/minions/<node name>/<app id>.<process type>.<process id>
//
// For example:
//
//     /empire/minions/ip-10-10-0-254/api.web.1
//     /empire/minions/ip-10-10-0-255/api.web.2
//
// The value of the key is a json encoded version of a Process
type consulRepository struct {
	client *api.Client
}

func NewConsulRepository(c *api.Client) *consulRepository {
	return &consulRepository{client: c}
}

func (c *consulRepository) HealthyMinions() (MinionMap, error) {
	entries, _, err := c.client.Health().Service(MinionsServiceName, "", true, nil)
	if err != nil {
		return MinionMap{}, err
	}

	minionmap := make(MinionMap, len(entries))

	for _, entry := range entries {
		n := Minion(*entry.Node)
		minionmap[Node(entry.Node.Node)] = &n
	}

	return minionmap, err
}

func (c *consulRepository) ScheduledMinions() (MinionMap, error) {
	nodes, _, err := c.client.KV().Keys(MinionsPrefix, "/", nil)
	if err != nil {
		return MinionMap{}, err
	}

	minionmap := make(MinionMap, len(nodes))

	for _, node := range nodes {
		minionmap[Node(node)] = &Minion{Node: node}
	}

	return minionmap, err
}

func (c *consulRepository) MinionSchedule(m *Minion) (Processes, error) {
	pairs, _, err := c.client.KV().List(c.minionKey(m), &api.QueryOptions{})
	if err != nil {
		return Processes{}, err
	}

	procs := make(Processes, len(pairs))
	for i, pair := range pairs {
		p, err := c.decode(pair)
		if err != nil {
			return Processes{}, err
		}

		procs[i] = p
	}

	return procs, nil
}

func (c *consulRepository) ProcessMap() (ProcessMap, error) {
	pairs, _, err := c.client.KV().List(MinionsPrefix, &api.QueryOptions{})
	if err != nil {
		return ProcessMap{}, err
	}

	procmap := ProcessMap{}
	for _, pair := range pairs {
		p, err := c.decode(pair)
		if err != nil {
			return ProcessMap{}, err
		}

		if _, ok := procmap[p.Name()]; !ok {
			procmap[p.Name()] = Processes{}
		}

		procmap[p.Name()] = append(procmap[p.Name()], p)
	}

	return procmap, nil
}

func (c *consulRepository) DeleteMinion(m *Minion) error {
	_, err := c.client.KV().DeleteTree(c.minionKey(m), nil)
	return err
}

func (c *consulRepository) DeleteProc(p Process) error {
	_, err := c.client.KV().Delete(c.procKey(p), nil)
	return err
}

func (c *consulRepository) procKey(p Process) string {
	return fmt.Sprintf("%s/%s.%s.%d", p.Minion.Node, p.AppID, p.ProcessType, p.ID)
}

func (c *consulRepository) minionKey(m *Minion) string {
	return fmt.Sprintf("%s/%s", MinionsPrefix, m.Node)
}

func (c *consulRepository) decode(pair *api.KVPair) (Process, error) {
	p := Process{}
	err := json.Unmarshal(pair.Value, &p)
	return p, err
}
