package heroku

import "github.com/remind101/empire/empire"

type mockEmpire struct {
	AccessTokensFindFunc func(string) (*empire.AccessToken, error)
}

func (e *mockEmpire) AccessTokensFind(token string) (*empire.AccessToken, error) {
	return e.AccessTokensFindFunc(token)
}
