package heroku

import (
	"net/http"
	"strconv"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/headerutil"
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

type GetRelease struct {
	*empire.Empire
}

func (h *GetRelease) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	vars := httpx.Vars(ctx)
	vers, err := strconv.Atoi(vars["version"])
	if err != nil {
		return err
	}

	rel, err := h.ReleasesFindByAppAndVersion(a, vers)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newRelease(rel))
}

type GetReleases struct {
	*empire.Empire
}

func (h *GetReleases) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	var rangeHeader headerutil.Range
	if header := r.Header.Get("Range"); header != "" {
		rangeHeader = headerutil.ParseRange(header)
	}

	l, err := strconv.Atoi(rangeHeader["max"])
	if err != nil {
		return err
	}

	rels, err := h.ReleasesFindByApp(a, &l)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newReleases(rels))
}

type PostReleases struct {
	*empire.Empire
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

func (h *PostReleases) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

	release, err := h.ReleasesRollback(ctx, app, version)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newRelease(release))
}
