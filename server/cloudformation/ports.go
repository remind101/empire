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

// InstancePortsResource is a Provisioner that allocates instance ports.
type InstancePortsResource struct {
	ports portAllocator
}

func newInstancePortsProvisioner(resource *InstancePortsResource) *provisioner {
	return &provisioner{
		Create: resource.Create,
		Delete: resource.Delete,
	}
}

func (p *InstancePortsResource) Properties() interface{} {
	return nil
}

func (p *InstancePortsResource) Create(_ context.Context, req customresources.Request) (string, interface{}, error) {
	var port int64
	port, err := p.ports.Get()
	data := map[string]int64{"InstancePort": port}
	id := fmt.Sprintf("%d", port)
	return id, data, err
}

func (p *InstancePortsResource) Delete(_ context.Context, req customresources.Request) error {
	port, err := strconv.Atoi(req.PhysicalResourceId)
	if err != nil {
		return fmt.Errorf("physical resource id should have been a port number: %v", err)
	}
	return p.ports.Put(int64(port))
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
