package units

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/consul/api"
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/releases"
)

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
