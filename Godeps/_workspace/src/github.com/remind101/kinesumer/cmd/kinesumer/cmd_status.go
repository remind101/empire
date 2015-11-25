package main

import (
	"time"

	"github.com/codegangsta/cli"
	"github.com/fatih/color"
	"github.com/remind101/kinesumer"
	"github.com/remind101/kinesumer/checkpointers/redis"
	"github.com/remind101/kinesumer/provisioners/redis"
	"github.com/remind101/kinesumer/redispool"
)

var cmdStatus = cli.Command{
	Name:    "status",
	Aliases: []string{"s"},
	Usage:   "Gets the status of a kinesis stream",
	Action:  runStatus,
	Flags:   append(flagsStream, flagsRedis...),
}

func runStatus(ctx *cli.Context) {
	k, err := kinesumer.NewDefault(
		getStream(ctx),
	)
	if err != nil {
		panic(err)
	}

	var prov *redisprovisioner.Provisioner
	var cp *redischeckpointer.Checkpointer
	redis := false
	if redisURL := ctx.String(fRedisURL); len(redisURL) > 0 {
		pool, err := redispool.NewRedisPool(redisURL)
		if err != nil {
			panic(err)
		}
		prefix := ctx.String(fRedisPrefix)

		prov, err = redisprovisioner.New(&redisprovisioner.Options{
			TTL:         time.Second,
			RedisPool:   pool,
			RedisPrefix: prefix,
		})
		if err != nil {
			panic(err)
		}

		cp, err = redischeckpointer.New(&redischeckpointer.Options{
			ReadOnly:    true,
			RedisPool:   pool,
			RedisPrefix: prefix,
		})

		err = cp.Begin()
		if err != nil {
			panic(err)
		}
		defer cp.End()

		redis = true
	}

	table := NewTable()
	header := table.AddRowWith("Shard ID", "Status")
	header.Header = true
	if redis {
		header.AddCellWithf("Worker")
		header.AddCellWithf("Sequence Number")
	}

	shards, err := k.GetShards()
	if err != nil {
		panic(err)
	}

	for _, shard := range shards {
		row := table.AddRow()
		row.AddCellWithf("%s", *shard.ShardId)
		if shard.SequenceNumberRange.EndingSequenceNumber == nil {
			row.AddCellWithf("OPEN").Color = color.New(color.FgGreen)
		} else {
			row.AddCellWithf("CLOSED").Color = color.New(color.FgRed)
		}
		if redis {
			cell := row.AddCell()
			lock, err := prov.Check(*shard.ShardId)
			if err != nil {
				lock = err.Error()
				cell.Color = color.New(color.FgRed)
			}
			cell.Printf("%s", lock)
			seqStart := StrShorten(cp.GetStartSequence(*shard.ShardId), 8, 8)
			cell = row.AddCell()
			if len(seqStart) == 0 {
				seqStart = "???"
				cell.Color = color.New(color.FgRed)
			}
			cell.Printf("%s", seqStart)
		}
	}

	table.Done()
}
