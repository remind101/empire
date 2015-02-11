package units

import (
	"errors"
	"fmt"
)

var (
	ErrNoReleaseFound = errors.New("No existing release found for repo")
)

type Release struct {
	Repo         string            `json:"repo"`
	ID           string            `json:"id"`
	Version      string            `json:"version"`
	Vars         map[string]string `json:"vars"`
	ProcessTypes map[string]string `json:"process_types"`
	ImageID      string            `json:"image_id"`
}

type ProcDef struct {
	Repo          string `json:"repo"`
	ReleaseID     string `json:"release_id"`
	ProcessType   string `json:"process_type"`
	InstanceCount int    `json:"instance_count"`
}

func NewProcDef(repo string, release string, proctype string, count int) ProcDef {
	return ProcDef{
		Repo:          repo,
		ReleaseID:     release,
		ProcessType:   proctype,
		InstanceCount: count,
	}
}

func (p ProcDef) Eql(other ProcDef) bool {
	return p.Repo == other.Repo &&
		p.ReleaseID == other.ReleaseID &&
		p.ProcessType == other.ProcessType &&
		p.InstanceCount == other.InstanceCount
}

func (p ProcDef) String() string {
	return fmt.Sprintf("%v/%v release=%v count=%v", p.Repo, p.ProcessType, p.ReleaseID, p.InstanceCount)
}

// Repository is an interface for storing a ProcDef
type Repository interface {
	Create(Release) error
	Patch(ProcDef) error
	Delete(ProcDef) error
	FindByRepo(string) ([]ProcDef, error)
}

type Service struct {
	Repository
}

func NewService(r Repository) *Service {
	return &Service{Repository: r}
}

// Create creates a new release for a repo
//
// If existing process definitions exist, they are updated with the new release
// Else a process definition is created for each process type in the release
// with an instance count of 0. The special case `web` will get an initial
// instance count of 1.
//
// Additionally, any process definitions that do not exist in this release's
// process types will be deleted.
func (s *Service) CreateRelease(rel Release) error {
	// Create the release
	err := s.Repository.Create(rel)
	if err != nil {
		return err
	}

	// Find existing process definitions
	defs, err := s.FindByRepo(rel.Repo)
	if err != nil {
		return err
	}

	// Create a process definition map
	defmap := make(map[string]ProcDef)
	for _, def := range defs {
		defmap[def.ProcessType] = def
	}

	// For each process type in new release,
	// update or create
	for pt := range rel.ProcessTypes {
		var def ProcDef
		var found bool

		if def, found = defmap[pt]; found {
			def.ReleaseID = rel.ID
		} else {
			count := 0
			if pt == "web" {
				count = 1
			}
			def = NewProcDef(rel.Repo, rel.ID, pt, count)
		}

		err := s.Repository.Patch(def)
		if err != nil {
			return err
		}
	}

	// For each existing process definition
	// If not included in new release's proc types, delete it
	for pt, def := range defmap {
		if _, found := rel.ProcessTypes[pt]; !found {
			err := s.Repository.Delete(def)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Patch updates a process definition
func (s *Service) Patch(repo string, proctype string, count int) (ProcDef, error) {
	var def ProcDef
	var err error

	defs, err := s.FindByRepo(repo)
	if err != nil {
		return def, err
	}

	if len(defs) == 0 {
		return def, ErrNoReleaseFound
	}

	def = NewProcDef(repo, defs[0].ReleaseID, proctype, count)
	err = s.Repository.Patch(def)
	if err != nil {
		return def, err
	}

	return def, nil
}

func (s *Service) FindByRepo(repo string) ([]ProcDef, error) {
	return s.Repository.FindByRepo(repo)
}

func (s *Service) Delete(repo string, proctype string) error {
	defs, err := s.FindByRepo(repo)
	if err != nil {
		return err
	}

	for _, def := range defs {
		if proctype == "" || proctype == def.ProcessType {
			err := s.Repository.Delete(def)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
