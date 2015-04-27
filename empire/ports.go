package empire

import (
	"database/sql"
	"errors"
)

type Port struct {
	ID    string  `db:"id"`
	AppID *string `db:"app_id"`
	Port  int     `db:"port"`
}

var ErrNoPorts = errors.New("no ports avaiable")

func (s *store) PortsFindOrCreateByApp(app *App) (*Port, error) {
	p, err := s.PortsFindByApp(app)

	// If an error occurred or we found a port, return.
	if err != nil || p != nil {
		return p, err
	}

	return s.PortsAssign(app)
}

func (s *store) PortsFindByApp(app *App) (*Port, error) {
	return portsFindByApp(s.db, app)
}

func (s *store) PortsAssign(app *App) (*Port, error) {
	var port *Port

	t, err := s.db.Begin()
	if err != nil {
		return port, err
	}

	port, err = portsFindAvailable(t)
	if err != nil {
		t.Rollback()
		return port, err
	}

	// Assign app to port
	port.AppID = &app.Name

	if _, err := portsUpdate(t, port); err != nil {
		t.Rollback()
		return port, err
	}

	return port, t.Commit()
}

func (s *store) PortsUnassign(app *App) error {
	_, err := portsUnassign(s.db, app)
	return err
}

func portsFindByApp(db *db, app *App) (*Port, error) {
	var port *Port
	err := db.SelectOne(&port, `select * from ports where app_id = $1 order by port limit 1`, app.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return port, err
}

func portsFindAvailable(db *Transaction) (*Port, error) {
	var port *Port
	err := db.SelectOne(&port, `select * from ports where app_id is null order by port limit 1`)
	if err == sql.ErrNoRows {
		return nil, ErrNoPorts
	}

	return port, err

}

func portsUpdate(db *Transaction, port *Port) (int64, error) {
	return db.Update(port)
}

func portsUnassign(db *db, app *App) (sql.Result, error) {
	return db.Exec(`update ports set app_id = null where app_id = $1`, app.Name)
}
