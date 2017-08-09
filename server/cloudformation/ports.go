package cloudformation

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/remind101/empire/pkg/cloudformation/customresources"

	"golang.org/x/net/context"
)

type portAllocator interface {
	Get() (int64, error)
	Put(port int64) error
}

// InstancePortsProvisioner is a Provisioner that allocates instance ports.
type InstancePortsProvisioner struct {
	ports portAllocator
}

func (p *InstancePortsProvisioner) Properties() interface{} {
	return nil
}

func (p *InstancePortsProvisioner) Provision(_ context.Context, req customresources.Request) (id string, data interface{}, err error) {
	switch req.RequestType {
	case customresources.Create:
		var port int64
		port, err = p.ports.Get()
		if err != nil {
			return
		}
		id = fmt.Sprintf("%d", port)
		data = map[string]int64{"InstancePort": port}
	case customresources.Delete:
		port, err2 := strconv.Atoi(req.PhysicalResourceId)
		if err2 != nil {
			err = fmt.Errorf("physical resource id should have been a port number: %v", err2)
			return
		}
		id = req.PhysicalResourceId
		err = p.ports.Put(int64(port))
	default:
		err = fmt.Errorf("%s is not supported", req.RequestType)
	}

	return
}

// dbPortAllocator implements the portAllocator interface backed by database/sql.
type dbPortAllocator struct {
	db *sql.DB
}

// NewdbPortAllocator returns a new dbPortAllocator uses the given database
// connection to perform queries.
func newDBPortAllocator(db *sql.DB) *dbPortAllocator {
	return &dbPortAllocator{db: db}
}

// Get finds an existing allocated port from the `ports` table. If one is not
// allocated for the process, it allocates one and returns it.
func (a *dbPortAllocator) Get() (int64, error) {
	sql := `UPDATE ports SET taken = true WHERE port = (SELECT port FROM ports WHERE taken IS NULL ORDER BY port ASC LIMIT 1) RETURNING port`
	var port int64
	err := a.db.QueryRow(sql).Scan(&port)
	return port, err
}

// Put releases any allocated port for the process, returning it back to the
// pool.
func (a *dbPortAllocator) Put(port int64) error {
	sql := `UPDATE ports SET taken = NULL WHERE port = $1`
	_, err := a.db.Exec(sql, port)
	return err
}
