package units

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/remind101/empire/apps"
)

type repository struct {
	sync.RWMutex
	units UnitMap
}

func newRepository() *repository {
	return &repository{
		units: make(UnitMap),
	}
}

func (r *repository) FindByName(n Name) (Unit, bool, error) {
	u, found := r.units[n]
	return u, found, nil
}

func (r *repository) FindByApp(id apps.ID) (UnitMap, error) {
	var unitmap = make(UnitMap)
	r.Lock()
	defer r.Unlock()

	for _, u := range r.units {
		if u.Release.App.ID == id {
			unitmap[u.Name()] = u
		}
	}

	return unitmap, nil
}

func (r *repository) FindAll() (UnitMap, error) {
	return r.units, nil
}

func (r *repository) Put(u Unit) error {
	r.Lock()
	defer r.Unlock()

	r.units[u.Name()] = u
	return nil
}

func (r *repository) Delete(u Unit) error {
	r.Lock()
	defer r.Unlock()

	delete(r.units, u.Name())
	return nil
}

type consulRepository struct {
	client *api.Client
}

func NewConsulRepository(c *api.Client) *consulRepository {
	return &consulRepository{client: c}
}

func (c *consulRepository) FindByName(n Name) (Unit, bool, error) {
	var u Unit

	pair, _, err := c.client.KV().Get(c.key(string(n)), &api.QueryOptions{})
	if err != nil {
		return u, false, err
	}

	u, err = c.decode(pair)
	if err != nil {
		return u, true, err
	}

	return u, true, nil
}

func (c *consulRepository) FindByApp(id apps.ID) (UnitMap, error) {
	var m UnitMap
	var err error

	pairs, _, err := c.client.KV().List(c.keyForApp(id), &api.QueryOptions{})
	if err != nil {
		return m, err
	}

	return c.decodeSlice(pairs)
}

func (c *consulRepository) FindAll() (UnitMap, error) {
	var m UnitMap
	var err error

	pairs, _, err := c.client.KV().List(c.key(""), &api.QueryOptions{})
	if err != nil {
		return m, err
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

func (c *consulRepository) keyForApp(id apps.ID) string {
	return c.key(string(id))
}

func (c *consulRepository) keyForUnit(u Unit) string {
	return c.key(string(u.Name()))
}

func (c *consulRepository) decodeSlice(pairs api.KVPairs) (UnitMap, error) {
	m := make(UnitMap)

	for _, pair := range pairs {
		u, err := c.decode(pair)
		if err != nil {
			return m, err
		}
		m[u.Name()] = u
	}

	return m, nil
}

func (c *consulRepository) decode(pair *api.KVPair) (Unit, error) {
	u := Unit{}
	err := json.Unmarshal(pair.Value, &u)
	return u, err
}

func (c *consulRepository) encode(u Unit) (*api.KVPair, error) {
	val, err := json.Marshal(&u)
	return &api.KVPair{Key: c.keyForUnit(u), Value: val}, err
}
