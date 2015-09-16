package redischeckpointer

import (
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/remind101/kinesumer/redispool"
)

var (
	prefix      = "testing"
	sequenceKey = prefix + ".sequence"
)

func makeCheckpointer() (*Checkpointer, error) {
	pool, err := redispool.NewRedisPool("redis://127.0.0.1:6379")
	if err != nil {
		return nil, err
	}
	r, err := New(&Options{
		SavePeriod:  time.Hour,
		RedisPool:   pool,
		RedisPrefix: prefix,
	})
	return r, err
}

func makeCheckpointerWithSamples() *Checkpointer {
	r, _ := makeCheckpointer()
	conn := r.pool.Get()
	defer conn.Close()
	conn.Do("DEL", sequenceKey)
	conn.Do("HSET", sequenceKey, "shard1", "1000")
	conn.Do("HSET", sequenceKey, "shard2", "2000")
	r, _ = makeCheckpointer()
	return r
}

func TestRedisGoodLogin(t *testing.T) {
	r, err := makeCheckpointer()
	if err != nil {
		t.Error("Failed to connect to redis at localhost:6379")
	}

	conn := r.pool.Get()
	defer conn.Close()
	reply, err := conn.Do("ECHO", "hey")

	re, err := redis.String(reply, err)
	if err != nil || re != "hey" {
		t.Error("Redis ECHO failed")
	}
}

func TestCheckpointerBeginEnd(t *testing.T) {
	r := makeCheckpointerWithSamples()
	err := r.Begin()
	if err != nil {
		t.Error(err)
	}
	r.End()
}

func TestCheckpointerGetStartSequence(t *testing.T) {
	r := makeCheckpointerWithSamples()
	_ = r.Begin()
	r.End()
	shard1 := "shard1"
	seq := r.GetStartSequence(shard1)
	if seq != "1000" {
		t.Error("Expected nonempty sequence number")
	}
}

func TestCheckpointerSync(t *testing.T) {
	r := makeCheckpointerWithSamples()
	r.Begin()
	r.heads["shard1"] = "1001"
	r.heads["shard2"] = "2001"
	r.Sync()
	r.End()
	r, _ = makeCheckpointer()
	r.Begin()
	r.End()
	if r.heads["shard1"] != "1001" {
		t.Error("Expected sequence number to be written")
	}
}
