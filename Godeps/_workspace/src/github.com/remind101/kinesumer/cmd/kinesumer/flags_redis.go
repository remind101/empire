package main

import (
	"github.com/codegangsta/cli"
)

var (
	fRedisURL    = "redis.url"
	fRedisPrefix = "redis.prefix"
)

var flagsRedis = []cli.Flag{
	cli.StringFlag{
		Name:   fRedisURL,
		Usage:  "The Redis URL",
		EnvVar: "REDIS_URL",
	},
	cli.StringFlag{
		Name:   fRedisPrefix,
		Usage:  "The Redis key prefix",
		EnvVar: "REDIS_PREFIX",
	},
}

func getRedisURL(ctx *cli.Context) string {
	return ctx.String("redis.url")
}
