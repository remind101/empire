package lb

import "database/sql"

// PortAllocator is an interface that allows us to allocate a port for an
// ELB.
//
// Because ELB can only forward to a single host port, we have to manage a list
// of host ports that we allocate to load balacers, to avoid any collisions.
type PortAllocator interface {
	// Get allocates a port from the pool.
	Get() (int64, error)
	// Put releases the allocated port back to the pool.
	Put(port int64) error
}

// DBPortAllocator implements the portAllocator interface backed by database/sql.
type DBPortAllocator struct {
	db *sql.DB
}

// NewDBPortAllocator returns a new DBPortAllocator uses the given database
// connection to perform queries.
func NewDBPortAllocator(db *sql.DB) *DBPortAllocator {
	return &DBPortAllocator{db: db}
}

// Get finds an existing allocated port from the `ports` table. If one is not
// allocated for the process, it allocates one and returns it.
func (a *DBPortAllocator) Get() (int64, error) {
	sql := `UPDATE ports SET taken = true WHERE port = (SELECT port FROM ports WHERE taken IS NULL ORDER BY port ASC LIMIT 1) RETURNING port`
	var port int64
	err := a.db.QueryRow(sql).Scan(&port)
	return port, err
}

// Put releases any allocated port for the process, returning it back to the
// pool.
func (a *DBPortAllocator) Put(port int64) error {
	sql := `UPDATE ports SET taken = NULL WHERE port = $1`
	_, err := a.db.Exec(sql, port)
	return err
}
