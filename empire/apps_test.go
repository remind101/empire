package empire

import (
	"testing"
)

func TestNewApp(t *testing.T) {
	_, err := NewApp("", "")
	if err != ErrInvalidName {
		t.Error("An empty name should be invalid")
	}

	a, err := NewApp("api", "remind101/r101-api")
	if err != nil {
		t.Fatal(err)
	}

	if want, got := AppName("api"), a.Name; want != got {
		t.Errorf("a.Name => %s; want %s", got, want)

	}
}

type mockAppsService struct {
	AppsService

	AppsFindOrCreateByRepoFunc func(repo Repo) (*App, error)
}

func (s *mockAppsService) AppsFindOrCreateByRepo(repo Repo) (*App, error) {
	if s.AppsFindOrCreateByRepoFunc != nil {
		return s.AppsFindOrCreateByRepoFunc(repo)
	}

	return nil, nil
}
