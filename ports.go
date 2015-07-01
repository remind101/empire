package empire

import (
	"errors"

	"github.com/jinzhu/gorm"
)

type Port struct {
	ID    string
	AppID *string
	Port  int
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

	t := s.db.Begin()

	port, err := portsFindAvailable(t)
	if err != nil {
		t.Rollback()
		return port, err
	}

	// Assign app to port
	port.AppID = &app.ID

	if err := portsUpdate(t, port); err != nil {
		t.Rollback()
		return port, err
	}

	if err := t.Commit().Error; err != nil {
		t.Rollback()
		return port, err
	}

	return port, nil
}

func (s *store) PortsUnassign(app *App) error {
	return portsUnassign(s.db, app)
}

func portsFindByApp(db *gorm.DB, app *App) (*Port, error) {
	var port Port
	if err := db.Where("app_id = ?", app.ID).Order("port").First(&port).Error; err != nil {
		if err == gorm.RecordNotFound {
			return nil, nil
		}

		return nil, err
	}
	return &port, nil
}

func portsFindAvailable(db *gorm.DB) (*Port, error) {
	var port Port
	if err := db.Where("app_id is null").Order("port").First(&port).Error; err != nil {
		if err == gorm.RecordNotFound {
			return nil, ErrNoPorts
		}

		return nil, err
	}
	return &port, nil
}

func portsUpdate(db *gorm.DB, port *Port) error {
	return db.Save(port).Error
}

func portsUnassign(db *gorm.DB, app *App) error {
	return db.Exec(`update ports set app_id = null where app_id = ?`, app.ID).Error
}
