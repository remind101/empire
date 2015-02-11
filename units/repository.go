package units

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/consul/api"
)

type consulRepository struct {
	client *api.Client
}

func NewConsulRepository(c *api.Client) *consulRepository {
	return &consulRepository{client: c}
}

func (c *consulRepository) Create(rel Release) error {
	val, err := json.Marshal(&rel)
	if err != nil {
		return err
	}

	pair := &api.KVPair{Key: c.keyForRelease(rel), Value: val}
	_, err = c.client.KV().Put(pair, nil)
	return err
}

func (c *consulRepository) FindByRepo(repo string) ([]ProcDef, error) {
	var err error

	pairs, _, err := c.client.KV().List(c.keyForProcDefs(repo), &api.QueryOptions{})
	if err != nil {
		return []ProcDef{}, err
	}

	return c.decodeSlice(pairs)
}

func (c *consulRepository) Patch(def ProcDef) error {
	pair, err := c.encode(def)
	if err != nil {
		return err
	}

	_, err = c.client.KV().Put(pair, nil)
	return err
}

func (c *consulRepository) Delete(def ProcDef) error {
	_, err := c.client.KV().Delete(c.keyForProcDef(def), nil)
	return err
}

// Private methods

func (c *consulRepository) key(k string) string {
	return fmt.Sprintf("empire/units/%s", k)
}

func (c *consulRepository) keyForRelease(rel Release) string {
	return c.key(fmt.Sprintf("releases/%v.%v", rel.Repo, rel.ID))
}

func (c *consulRepository) keyForProcDefs(repo string) string {
	return c.key(fmt.Sprintf("processes/%v", repo))
}

func (c *consulRepository) keyForProcDef(def ProcDef) string {
	return c.key(fmt.Sprintf("processes/%v.%v", def.Repo, def.ProcessType))
}

func (c *consulRepository) decodeSlice(pairs api.KVPairs) ([]ProcDef, error) {
	var err error
	defs := make([]ProcDef, len(pairs))

	for i, pair := range pairs {
		defs[i], err = c.decode(pair)
		if err != nil {
			return defs, err
		}
	}

	return defs, nil
}

func (c *consulRepository) decode(pair *api.KVPair) (ProcDef, error) {
	def := ProcDef{}
	err := json.Unmarshal(pair.Value, &def)
	return def, err
}

func (c *consulRepository) encode(def ProcDef) (*api.KVPair, error) {
	val, err := json.Marshal(&def)
	return &api.KVPair{Key: c.keyForProcDef(def), Value: val}, err
}
