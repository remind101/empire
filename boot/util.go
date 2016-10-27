package boot

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/remind101/empire"
)

// uriContentOrValue uses the following algorithm:
//
// 1. If the input is a URI, it will use uriContent to fetch the content from
// the URI with the proper scheme.
// 2. If the input is not a URI, it assumes that the value is the raw content,
// and returns it.
func uriContentOrValue(maybeURI string) ([]byte, error) {
	uri, err := url.Parse(maybeURI)
	if err != nil || uri.Scheme == "" {
		return []byte(maybeURI), nil
	}

	return uriContent(uri)
}

// uriContent fetches the content from the URI. It supports http://, https://
// and file:// schemes.
func uriContent(uri *url.URL) ([]byte, error) {
	// TODO: Support file://
	scheme := uri.Scheme
	switch scheme {
	case "https", "http":
		req, err := http.NewRequest("GET", uri.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", fmt.Sprintf("Empire (%s)", empire.Version))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return ioutil.ReadAll(resp.Body)
	case "file":
		return ioutil.ReadFile(uri.Path)
	default:
		return nil, fmt.Errorf("not able to fetch content from %s via %s", uri, scheme)
	}
}
