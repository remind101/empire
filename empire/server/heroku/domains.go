package heroku

import (
	"fmt"
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/empire"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

type Domain heroku.Domain

func newDomain(d *empire.Domain) *Domain {
	return &Domain{
		Id:        d.ID,
		Hostname:  d.Hostname,
		CreatedAt: *d.CreatedAt,
	}
}

type GetDomains struct {
	*empire.Empire
}

func (h *GetDomains) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	d, err := h.Domains(empire.DomainsQuery{App: a})
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, d)
}

type PostDomainsForm struct {
	Hostname string `json:"hostname"`
}

type PostDomains struct {
	*empire.Empire
}

func (h *PostDomains) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	var form PostDomainsForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	domain := &empire.Domain{
		AppID:    a.ID,
		Hostname: form.Hostname,
	}
	d, err := h.DomainsCreate(domain)
	if err != nil {
		if err == empire.ErrDomainInUse {
			return fmt.Errorf("%s is currently in use by another app.", domain.Hostname)
		} else if err == empire.ErrDomainAlreadyAdded {
			return fmt.Errorf("%s is already added to this app.", domain.Hostname)
		}
		return err
	}

	w.WriteHeader(201)
	return Encode(w, newDomain(d))
}

type DeleteDomain struct {
	*empire.Empire
}

func (h *DeleteDomain) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	vars := httpx.Vars(ctx)
	name := vars["hostname"]

	d, err := h.DomainsFirst(empire.DomainsQuery{Hostname: &name, App: a})
	if err != nil {
		if err == gorm.RecordNotFound {
			return &ErrorResource{
				Status:  http.StatusNotFound,
				ID:      "not_found",
				Message: "Couldn't find that domain name.",
			}
		}
		return err
	}

	if err = h.DomainsDestroy(d); err != nil {
		return err
	}

	return NoContent(w)
}
