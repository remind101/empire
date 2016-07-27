package heroku

import (
	"net/http"
	"strconv"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

type Release heroku.Release

func newRelease(r *empire.Release) *Release {
	return &Release{
		Id:      r.ID,
		Version: r.Version,
		Slug: &struct {
			Id string `json:"id"`
		}{
			Id: r.SlugID,
		},
		Description: r.Description,
		CreatedAt:   *r.CreatedAt,
	}
}

func newReleases(rs []*empire.Release) []*Release {
	releases := make([]*Release, len(rs))

	for i := 0; i < len(rs); i++ {
		releases[i] = newRelease(rs[i])
	}

	return releases
}

func (h *Server) GetRelease(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	vars := httpx.Vars(ctx)
	vers, err := strconv.Atoi(vars["version"])
	if err != nil {
		return err
	}

	rel, err := h.ReleasesFind(empire.ReleasesQuery{App: a, Version: &vers})
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newRelease(rel))
}

func (h *Server) GetReleases(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	rangeHeader, err := RangeHeader(r)
	if err != nil {
		return err
	}

	rels, err := h.Releases(empire.ReleasesQuery{App: a, Range: rangeHeader})
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newReleases(rels))
}

type PostReleasesForm struct {
	Version string `json:"release"`
}

func (p *PostReleasesForm) ReleaseVersion() (int, error) {
	vers, err := strconv.Atoi(p.Version)
	if err != nil {
		return vers, err
	}

	return vers, nil
}

func (h *Server) PostReleases(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PostReleasesForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	version, err := form.ReleaseVersion()
	if err != nil {
		return err
	}

	app, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	m, err := findMessage(r)
	if err != nil {
		return err
	}

	release, err := h.Rollback(ctx, empire.RollbackOpts{
		User:    UserFromContext(ctx),
		App:     app,
		Version: version,
		Message: m,
	})
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newRelease(release))
}
