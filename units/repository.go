package units

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/releases"
)

type repository struct {
	sync.RWMutex
	releases map[apps.ID]*releases.Release
	units    map[apps.ID]UnitMap
}

func newRepository() *repository {
	return &repository{
		releases: make(map[apps.ID]*releases.Release),
		units:    make(map[apps.ID]UnitMap),
	}
}

func (r *repository) Create(rel *releases.Release) error {
	r.Lock()
	defer r.Unlock()

	r.releases[rel.App.ID] = rel
	return nil
}

func (r *repository) Put(u Unit) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.units[u.Release.App.ID]; !ok {
		r.units[u.Release.App.ID] = make(UnitMap)
	}

	r.units[u.Release.App.ID][u.ProcessType] = u
	return nil
}

func (r *repository) Delete(u Unit) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.units[u.Release.App.ID]; !ok {
		return nil
	}

	delete(r.units[u.Release.App.ID], u.ProcessType)
	return nil
}

func (r *repository) FindByApp(id apps.ID) ([]Unit, error) {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.units[id]; !ok {
		return []Unit{}, nil
	}

	var units []Unit
	m := r.units[id]
	for _, u := range m {
		units = append(units, u)
	}

	return units, nil
}

type consulRepository struct {
	client *api.Client
}

func NewConsulRepository(c *api.Client) *consulRepository {
	return &consulRepository{client: c}
}

func (c *consulRepository) Create(rel *releases.Release) error {
	val, err := json.Marshal(rel)
	if err != nil {
		return err
	}

	pair := &api.KVPair{Key: c.keyForRelease(rel), Value: val}
	_, err = c.client.KV().Put(pair, nil)
	return err
}

func (c *consulRepository) FindByApp(id apps.ID) ([]Unit, error) {
	var err error

	pairs, _, err := c.client.KV().List(c.keyForUnits(string(id)), &api.QueryOptions{})
	if err != nil {
		return []Unit{}, err
	}

	return c.decodeSlice(pairs)
}

func (c *consulRepository) Put(u Unit) error {
	pair, err := c.encode(u)
	if err != nil {
		return err
	}

	_, err = c.client.KV().Put(pair, nil)
	return err
}

func (c *consulRepository) Delete(u Unit) error {
	_, err := c.client.KV().Delete(c.keyForUnit(u), nil)
	return err
}

// Private methods

func (c *consulRepository) key(k string) string {
	return fmt.Sprintf("empire/units/%s", k)
}

func (c *consulRepository) keyForRelease(rel *releases.Release) string {
	return c.key(fmt.Sprintf("releases/%v.%v", rel.App.ID, rel.ID))
}

func (c *consulRepository) keyForUnits(repo string) string {
	return c.key(fmt.Sprintf("processes/%v", repo))
}

func (c *consulRepository) keyForUnit(u Unit) string {
	return c.key(fmt.Sprintf("processes/%v.%v", u.Release.App.ID, u.ProcessType))
}

func (c *consulRepository) decodeSlice(pairs api.KVPairs) ([]Unit, error) {
	var err error
	defs := make([]Unit, len(pairs))

	for i, pair := range pairs {
		defs[i], err = c.decode(pair)
		if err != nil {
			return defs, err
		}
	}

	return defs, nil
}

func (c *consulRepository) decode(pair *api.KVPair) (Unit, error) {
	def := Unit{}
	err := json.Unmarshal(pair.Value, &def)
	return def, err
}

func (c *consulRepository) encode(def Unit) (*api.KVPair, error) {
	val, err := json.Marshal(&def)
	return &api.KVPair{Key: c.keyForUnit(def), Value: val}, err
}
