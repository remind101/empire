package main

import "github.com/remind101/empire/boot"

func runServer(c *Context) {
	ctx, err := bootContext(c)
	c.must(err)

	e, err := boot.BootContext(ctx)
	c.must(err)

	c.must(e.Start())
}
