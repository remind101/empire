package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/inconshreveable/log15/term"
	"github.com/mattn/go-colorable"
	"github.com/remind101/empire"
	"github.com/remind101/empire/boot"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

var (
	flagConfig = cli.StringFlag{
		Name:   "config",
		Value:  "/etc/empire.toml",
		Usage:  "Path to the empire.toml config file",
		EnvVar: "EMPIRE_CONFIG",
	}
	flagConfigNoValidate = cli.BoolFlag{
		Name:  "config.novalidate",
		Usage: "Disable validation of the config. Only use this if you know what you're doing!",
	}
	flagLogLevel = cli.StringFlag{
		Name:   "log.level",
		Value:  "info",
		Usage:  "Specify the log level for the empire server. You can use this to enable debug logs by specifying `debug`.",
		EnvVar: "EMPIRE_LOG_LEVEL",
	}
)

var flags = []cli.Flag{
	flagConfig,
	flagConfigNoValidate,
	flagLogLevel,
}

// Commands are the subcommands that are available.
var Commands = []cli.Command{
	{
		Name:      "server",
		ShortName: "s",
		Usage:     "Run the empire HTTP api",
		Flags:     flags,
		Action:    withLogger(runServer),
	},
	{
		Name:   "migrate",
		Usage:  "Migrate the database",
		Flags:  flags,
		Action: withLogger(runMigrate),
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "empire"
	app.Usage = "Platform as a Binary"
	app.Version = empire.Version
	app.Commands = Commands

	app.Run(os.Args)
}

func bootContext(c *Context) (*boot.Context, error) {
	config, err := parseConfig(c)
	if err != nil {
		return nil, err
	}

	return boot.NewRootContext(c, config)
}

func parseConfig(c *Context) (*boot.Config, error) {
	path := c.String(flagConfig.Name)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config, err := boot.ParseConfig(f)
	if err != nil {
		return config, fmt.Errorf("unable to parse %s: %v", f.Name(), err)
	}

	if !c.Bool(flagConfigNoValidate.Name) {
		if err := boot.ValidateProductionConfig(config); err != nil {
			return config, fmt.Errorf("%s: %v", f.Name(), err)
		}
	}

	return config, nil
}

type netCtx context.Context

// Context wraps a cli.Context and a net.Context as a single unit.
type Context struct {
	*cli.Context
	netCtx
}

func (ctx *Context) must(err error) {
	if err != nil {
		logger.Crit(ctx, err.Error())
		os.Exit(1)
	}
}

func withLogger(f func(*Context)) func(*cli.Context) {
	return func(c *cli.Context) {
		ctx := logger.WithLogger(context.Background(), newLogger(c))
		f(&Context{Context: c, netCtx: ctx})
	}
}

func newLogger(c *cli.Context) log15.Logger {
	l := log15.New()

	lvl, err := log15.LvlFromString(c.String(flagLogLevel.Name))
	if err != nil {
		panic(err)
	}

	var stdout log15.Handler
	if term.IsTty(os.Stdout.Fd()) {
		stdout = log15.StreamHandler(colorable.NewColorableStdout(), log15.TerminalFormat())
	} else {
		stdout = log15.StreamHandler(os.Stdout, log15.LogfmtFormat())
	}

	h := log15.LvlFilterHandler(lvl, stdout)
	if lvl == log15.LvlDebug {
		h = log15.CallerFileHandler(h)
	}

	l.SetHandler(log15.LazyHandler(h))
	return l
}
