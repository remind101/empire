package main

import (
	"github.com/remind101/empire/boot"
	"github.com/remind101/pkg/logger"
)

func runMigrate(c *Context) {
	ctx, err := bootContext(c)
	c.must(err)

	c.must(boot.MigrateUp(ctx))
	logger.Info(c, "Up to date")
}
