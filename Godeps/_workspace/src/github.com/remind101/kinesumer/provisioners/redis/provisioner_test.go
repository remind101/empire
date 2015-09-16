package redisprovisioner

import (
	"testing"
	"time"

	"github.com/remind101/kinesumer/redispool"
	"github.com/stretchr/testify/assert"
)

func makeProvisioner() *Provisioner {
	pool, err := redispool.NewRedisPool("redis://127.0.0.1:6379")
	if err != nil {
		panic(err)
	}

	conn := pool.Get()
	defer conn.Close()

	conn.Do("DEL", "testing:lock:shard0")

	prov, err := New(&Options{
		TTL:         time.Second,
		RedisPool:   pool,
		RedisPrefix: "testing",
		Lock:        "lock",
	})
	if err != nil {
		panic(err)
	}

	return prov
}

func TestProvisionerTryAcquire(t *testing.T) {
	p := makeProvisioner()
	assert.NoError(t, p.TryAcquire("shard0"), "Couldn't acquire lock")

	assert.Error(t, p.TryAcquire("shard0"), "Acquired lock")
}

func TestProvisionerRelease(t *testing.T) {
	p := makeProvisioner()
	assert.NoError(t, p.TryAcquire("shard0"), "Couldn't acquire lock")

	assert.NoError(t, p.Release("shard0"), "Couldn't release lock")

	assert.NoError(t, p.TryAcquire("shard0"), "Couldn't reacquire lock")
}

func TestProvisionerHeartbeat(t *testing.T) {
	p := makeProvisioner()
	err := p.Heartbeat("shard0")
	assert.Error(t, err, "managed to heartbeat without acquiring lock")

	assert.NoError(t, p.TryAcquire("shard0"), "Couldn't acquire lock")

	assert.NoError(t, p.Heartbeat("shard0"), "Couldn't heartbeat")

	assert.Equal(t, 1, len(p.heartbeats))
}
