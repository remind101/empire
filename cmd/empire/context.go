package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/codegangsta/cli"
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

	return c.awsConfigProvider.ClientConfig(serviceName)
}
