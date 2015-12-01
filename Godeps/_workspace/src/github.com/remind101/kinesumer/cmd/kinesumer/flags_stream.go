package main

import (
	"github.com/codegangsta/cli"
)

var flagsStream = []cli.Flag{
	cli.StringFlag{
		Name:  "stream, s",
		Usage: "The Kinesis stream to tail",
	},
}

func getStream(ctx *cli.Context) string {
	return ctx.String("stream")
}
