package units

import (
	"testing"

	"github.com/remind101/empire/consulutil"
)

func TestConsulCreate(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	service := NewConsulRepository(c)

	err := service.Create(Release{
		Repo:    "api",
		ID:      "1",
		Version: "v1",
		Vars: map[string]string{
			"RAILS_ENV": "production",
		},
		ProcessTypes: map[string]string{
			"web": "./bin/web",
		},
		ImageID: "abcd",
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestConsulFindByRepo(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	service := NewConsulRepository(c)

	// Add some data
	if err := service.Patch(NewProcDef("api", "v1", "web", 3)); err != nil {
		t.Fatal(err)
	}
	if err := service.Patch(NewProcDef("api", "v1", "worker", 6)); err != nil {
		t.Fatal(err)
	}

	if err := service.Patch(NewProcDef("dash", "v1", "web", 9)); err != nil {
		t.Fatal(err)
	}

	testProcDefsEql(t, service, "api", []ProcDef{
		NewProcDef("api", "v1", "web", 3),
		NewProcDef("api", "v1", "worker", 6),
	})

	testProcDefsEql(t, service, "dash", []ProcDef{
		NewProcDef("dash", "v1", "web", 9),
	})
}

func TestConsulPatch(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	service := NewConsulRepository(c)

	var err error

	// Add some data
	if err = service.Patch(NewProcDef("api", "v1", "web", 3)); err != nil {
		t.Fatal(err)
	}

	testProcDefsEql(t, service, "api", []ProcDef{
		NewProcDef("api", "v1", "web", 3),
	})

	// Update existing data
	if err = service.Patch(NewProcDef("api", "v1", "web", 10)); err != nil {
		t.Fatal(err)
	}

	testProcDefsEql(t, service, "api", []ProcDef{
		NewProcDef("api", "v1", "web", 10),
	})
}

func TestConsulDelete(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	service := NewConsulRepository(c)

	var err error

	if err = service.Patch(NewProcDef("api", "v1", "web", 3)); err != nil {
		t.Fatal(err)
	}

	testProcDefsEql(t, service, "api", []ProcDef{
		NewProcDef("api", "v1", "web", 3),
	})

	if err = service.Delete(NewProcDef("api", "v1", "web", 3)); err != nil {
		t.Fatal(err)
	}

	testProcDefsEql(t, service, "api", []ProcDef{})
}
