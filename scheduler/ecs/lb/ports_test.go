package lb

import (
	"testing"

	_ "github.com/lib/pq"
	"github.com/remind101/empire/dbtest"
	"github.com/stretchr/testify/assert"
)

func TestDBPortAllocator_Get(t *testing.T) {
	db := dbtest.Open(t)
	a := &DBPortAllocator{
		db: db,
	}

	port, err := a.Get()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, port)

	err = a.Put(port)
	assert.NoError(t, err)
}
