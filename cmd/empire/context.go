package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/saml"
	"github.com/remind101/empire/stats"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

type netCtx context.Context

// Context provides lazy loaded, memoized instances of services the CLI
// consumes. It also implements the context.Context interfaces with embedded
// reporter.Repoter, logger.Logger, and stats.Stats implementations, so it can
// be injected as a top level context object.
type Context struct {
	*cli.Context
	netCtx

	// Error reporting, logging and stats.
	reporter reporter.Reporter
	logger   logger.Logger
	stats    stats.Stats

	// AWS stuff
	awsConfigProvider client.ConfigProvider

	samlServiceProvider *saml.ServiceProvider
}

// newContext builds a new base Context object.
func newContext(c *cli.Context) (ctx *Context, err error) {
	ctx = &Context{
		Context: c,
		netCtx:  context.Background(),
	}

	ctx.reporter, err = newReporter(ctx)
	if err != nil {
		return
	}

	ctx.logger, err = newLogger(ctx)
	if err != nil {
		return
	}

	ctx.stats, err = newStats(ctx)
	if err != nil {
		return
	}

	if ctx.reporter != nil {
		ctx.netCtx = reporter.WithReporter(ctx.netCtx, ctx.reporter)
	}
	if ctx.logger != nil {
		ctx.netCtx = logger.WithLogger(ctx.netCtx, ctx.logger)
	}
	if ctx.stats != nil {
		ctx.netCtx = stats.WithStats(ctx.netCtx, ctx.stats)
	}

	return
}

func (c *Context) URL(name string) *url.URL {
	v := c.String(name)
	u, err := url.Parse(v)
	if err != nil {
		panic(err)
	}
	return u
}

func (c *Context) Reporter() reporter.Reporter { return c.reporter }
func (c *Context) Logger() logger.Logger       { return c.logger }
func (c *Context) Stats() stats.Stats          { return c.stats }

// ClientConfig implements the client.ConfigProvider interface. This will return
// a mostly standard client.Config, but also includes middleware that will
// generate metrics for retried requests, and enables debug mode if
// `FlagAWSDebug` is set.
func (c *Context) ClientConfig(serviceName string, cfgs ...*aws.Config) client.Config {
	if c.awsConfigProvider == nil {
		c.awsConfigProvider = newConfigProvider(c)
	}

	return c.awsConfigProvider.ClientConfig(serviceName, cfgs...)
}

func (c *Context) SAMLServiceProvider() (*saml.ServiceProvider, error) {
	if c.samlServiceProvider == nil {
		metadataLocation := c.String(FlagSAMLMetadata)
		if metadataLocation == "" {
			// No SAML
			return nil, nil
		}

		metadataContent, err := uriContentOrValue(metadataLocation)
		if err != nil {
			return nil, err
		}

		baseURL := c.URL(FlagURL)

		var metadata saml.Metadata
		if err := xml.Unmarshal(metadataContent, &metadata); err != nil {
			return nil, fmt.Errorf("error parsing SAML metadata: %v", err)
		}

		c.samlServiceProvider = &saml.ServiceProvider{
			IDPMetadata: &metadata,
			MetadataURL: fmt.Sprintf("%s/saml/metadata", baseURL),
			AcsURL:      fmt.Sprintf("%s/saml/acs", baseURL),
		}

		if v := c.String(FlagSAMLKey); v != "" {
			key, err := uriContentOrValue(c.String(FlagSAMLKey))
			if err != nil {
				return nil, err
			}
			cert, err := uriContentOrValue(c.String(FlagSAMLCert))
			if err != nil {
				return nil, err
			}
			c.samlServiceProvider.Key = string(key)
			c.samlServiceProvider.Certificate = string(cert)
		}
	}

	return c.samlServiceProvider, nil
}

// uriContentOrValue uses the following algorithm:
//
// 1. If the input is a URI, it will use uriContent to fetch the content from
// the URI with the proper scheme.
// 2. If the input is not a URI, it assumes that the value is the raw content,
// and returns it.
func uriContentOrValue(maybeURI string) ([]byte, error) {
	uri, err := url.Parse(maybeURI)
	if err != nil {
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
		return nil, fmt.Errorf("not able to fetch metadata via %s", scheme)
	}
}
