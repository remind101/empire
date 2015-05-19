package empire

import "testing"

func TestCertificatesQuery(t *testing.T) {
	id := "1234"
	app := &App{ID: "4321"}

	tests := scopeTests{
		{CertificatesQuery{}, "", []interface{}{}},
		{CertificatesQuery{ID: &id}, "WHERE (id = $1)", []interface{}{id}},
		{CertificatesQuery{App: app}, "WHERE (app_id = $1)", []interface{}{app.ID}},
	}

	tests.Run(t)
}
