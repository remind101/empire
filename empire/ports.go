package empire

import "database/sql"

type Port struct {
	ID    string `db:"id"`
	AppID string `db:"app_id"`
	Port  int    `db:"port"`
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
		return port, err
	}

	// Assign app to port
	port.AppID = app.Name

	if _, err := portsUpdate(t, port); err != nil {
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
	return port, db.SelectOne(&port, `select * from ports where app_id = $1 order by port`, app.Name)
}

func portsFindAvailable(db *Transaction) (*Port, error) {
	var port *Port
	return port, db.SelectOne(&port, `select * from ports where app_id is null order by port`)
}

func portsUpdate(db *Transaction, port *Port) (int64, error) {
	return db.Update(port)
}

func portsUnassign(db *db, app *App) (sql.Result, error) {
	return db.Exec(`update ports set app_id = null where app_id = $1`, app.Name)
}
