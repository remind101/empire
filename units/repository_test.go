package units

import (
	"testing"

	"github.com/remind101/empire/consulutil"
	"github.com/remind101/empire/slugs"
)

func TestConsulCreate(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	service := NewConsulRepository(c)

	err := service.Create(buildRelease("api", "1", slugs.ProcessMap{"web": "./bin/web"}))

	if err != nil {
		t.Fatal(err)
	}
}

func TestConsulFindByApp(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	service := NewConsulRepository(c)

	// Add some data
	rel := buildRelease("api", "1", slugs.ProcessMap{
		"web":      "./bin/web",
		"worker":   "./bin/worker",
		"consumer": "./bin/consumer",
	})
	service.Create(rel)
	service.Put(NewUnit(rel, "web", 10))
	service.Put(NewUnit(rel, "worker", 5))
	service.Put(NewUnit(rel, "consumer", 3))

	units, err := service.FindByApp("api")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(units), 3; got != want {
		t.Errorf("len(service.FindByApp(\"api\")) => %s; want %s", got, want)
	}
}

func TestConsulPut(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	service := NewConsulRepository(c)

	var err error

	// Add some data using a release
	rel := buildRelease("api", "1", slugs.ProcessMap{
		"web":      "./bin/web",
		"worker":   "./bin/worker",
		"consumer": "./bin/consumer",
	})

	if err = service.Put(NewUnit(rel, "web", 10)); err != nil {
		t.Fatal(err)
	}

	testUnitsEql(t, service, "api", []string{
		"api.web release=1 count=10",
	})
}

func TestConsulDelete(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	service := NewConsulRepository(c)

	var err error

	// Add some data
	rel := buildRelease("api", "1", slugs.ProcessMap{
		"web":      "./bin/web",
		"worker":   "./bin/worker",
		"consumer": "./bin/consumer",
	})
	service.Create(rel)
	service.Put(NewUnit(rel, "web", 10))
	service.Put(NewUnit(rel, "worker", 5))
	service.Put(NewUnit(rel, "consumer", 3))

	testUnitsEql(t, service, "api", []string{
		"api.web release=1 count=10",
		"api.worker release=1 count=5",
		"api.consumer release=1 count=3",
	})

	if err = service.Delete(NewUnit(rel, "web", 0)); err != nil {
		t.Fatal(err)
	}

	testUnitsEql(t, service, "api", []string{
		"api.worker release=1 count=5",
		"api.consumer release=1 count=3",
	})
}
