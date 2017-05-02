package config

import (
	"fmt"
	"net/url"

	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb2"
	"github.com/remind101/pkg/reporter/rollbar"
)

// Returns a MultiReporter from URL strings such as:
// "hb://api.honeybadger.io/?key=hbkey&environment=hbenv" or
// "rollbar://api.rollbar.com/?key=rollbarkey&environment=rollbarenv"
func NewReporterFromUrls(urls []string) (reporter.Reporter, error) {
	multiRep := reporter.MultiReporter{}
	for _, url := range urls {
		rep, err := newReporterFromUrl(url)
		if err != nil {
			return nil, err
		}
		multiRep = append(multiRep, rep)
	}
	return multiRep, nil
}

func newReporterFromUrl(u string) (reporter.Reporter, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse %s: %#v", u, err)
	}
	switch parsedURL.Scheme {
	case "hb":
		q := parsedURL.Query()
		return hb2.NewReporter(hb2.Config{
			ApiKey:      q.Get("key"),
			Environment: q.Get("environment"),
		}), nil
	case "rollbar":
		q := parsedURL.Query()
		rollbar.ConfigureReporter(q.Get("key"), q.Get("environment"))
		return rollbar.Reporter, nil
	default:
		return nil, fmt.Errorf("unrecognized reporter url scheme: %s", u)
	}
}
