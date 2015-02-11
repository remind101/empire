package units

import (
	"sort"
	"testing"

	"github.com/remind101/empire/consulutil"
)

func TestCreate(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	ps := NewService(NewConsulRepository(c))

	// Creates ProcDefs for each process type
	err := ps.CreateRelease(Release{
		Repo:    "api",
		ID:      "1",
		Version: "v1",
		ImageID: "abc",
		ProcessTypes: map[string]string{
			"web":      "./web",
			"worker":   "./worker",
			"consumer": "./consumer",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	testProcDefsEql(t, ps.Repository, "api", []ProcDef{
		NewProcDef("api", "1", "web", 1),
		NewProcDef("api", "1", "worker", 0),
		NewProcDef("api", "1", "consumer", 0),
	})

	// Updates ProcDefs for each process type, removes any that don't exist anymore
	err = ps.CreateRelease(Release{
		Repo:    "api",
		ID:      "2",
		Version: "v2",
		ImageID: "abc",
		ProcessTypes: map[string]string{
			"web":    "./web",
			"worker": "./worker",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	testProcDefsEql(t, ps.Repository, "api", []ProcDef{
		NewProcDef("api", "2", "web", 1),
		NewProcDef("api", "2", "worker", 0),
	})
}

func TestPatch(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	ps := NewService(NewConsulRepository(c))
	var def ProcDef

	// web: 0 -> 3
	err := ps.CreateRelease(Release{
		Repo: "api", ID: "1", Version: "v1", ImageID: "abc", ProcessTypes: map[string]string{"web": "./web"},
	})
	if err != nil {
		t.Fatal(err)
	}

	def, err = ps.Patch("api", "web", 3)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := NewProcDef("api", "1", "web", 3).Eql(def), true; got != want {
		t.Fatalf("ProcDef.Eql() => %s; want %s", got, want)
	}
}

func TestDelete(t *testing.T) {
	c, s := consulutil.MakeClient(t)
	defer s.Stop()
	ps := NewService(NewConsulRepository(c))

	err := ps.CreateRelease(Release{
		Repo: "api", ID: "1", Version: "v1", ImageID: "abc", ProcessTypes: map[string]string{"web": "./web"},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = ps.Patch("api", "web", 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ps.Patch("api", "worker", 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ps.Patch("api", "consumer", 1)
	if err != nil {
		t.Fatal(err)
	}

	// Delete just one process type
	err = ps.Delete("api", "consumer")
	if err != nil {
		t.Fatal(err)
	}

	defs, err := ps.FindByRepo("api")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(defs), 2; got != want {
		t.Fatalf("len(defs => %s; want %s", got, want)
	}

	// Delete all processes for a repo
	err = ps.Delete("api", "")
	if err != nil {
		t.Fatal(err)
	}

	defs, err = ps.FindByRepo("api")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(defs), 0; got != want {
		t.Fatalf("len(defs => %s; want %s", got, want)
	}
}

func testProcDefsEql(t *testing.T, s Repository, repo string, defs []ProcDef) {
	foundDefs, err := s.FindByRepo(repo)
	if err != nil {
		t.Fatal(err)
	}

	sortedFoundDefs := byString(foundDefs)
	sortedDefs := byString(defs)

	sort.Sort(sortedFoundDefs)
	sort.Sort(sortedDefs)

	if got, want := len(sortedFoundDefs), len(sortedDefs); got != want {
		t.Errorf("len(s.FindByRepo(\"%s\")) => %v; want %v", repo, got, want)
	}

	for i, def := range sortedDefs {
		if got, want := sortedFoundDefs[i], def; !got.Eql(want) {
			t.Errorf("s.FindByRepo(\"%s\")[%v] => %v; want %v", repo, i, got, want)
		}
	}
}

type byString []ProcDef

func (b byString) Len() int {
	return len(b)
}

func (b byString) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byString) Less(i, j int) bool {
	return b[i].String() < b[j].String()
}
