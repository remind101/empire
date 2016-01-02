package redisprovisioner

import (
	"errors"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/pborman/uuid" // Exported from code.google.com/p/go-uuid/uuid
)

type Provisioner struct {
	acquired      map[string]bool
	heartbeats    map[string]time.Time
	heartbeatsMut sync.RWMutex
	ttl           time.Duration
	pool          *redis.Pool
	redisPrefix   string
	lock          string
}

type Options struct {
	TTL         time.Duration
	Lock        string
	RedisPool   *redis.Pool
	RedisPrefix string
}

func New(opt *Options) (*Provisioner, error) {
	if opt.Lock == "" {
		opt.Lock = uuid.New()
	}

	return &Provisioner{
		acquired:    make(map[string]bool),
		heartbeats:  make(map[string]time.Time),
		ttl:         opt.TTL,
		lock:        opt.Lock,
		pool:        opt.RedisPool,
		redisPrefix: opt.RedisPrefix,
	}, nil
}

func (p *Provisioner) TryAcquire(shardID string) error {
	if len(shardID) == 0 {
		return errors.New("ShardId cannot be empty")
	}

	conn := p.pool.Get()
	defer conn.Close()

	res, err := conn.Do("SET", p.redisPrefix+":lock:"+shardID, p.lock, "PX", int64(p.ttl/time.Millisecond), "NX")
	if err != nil {
		return err
	}
	if res != "OK" {
		return errors.New("Failed to acquire lock")
	}

	p.acquired[shardID] = true
	return nil
}

func (p *Provisioner) Release(shardID string) error {
	conn := p.pool.Get()
	defer conn.Close()

	delete(p.acquired, shardID)

	key := p.redisPrefix + ":lock:" + shardID
	res, err := redis.String(conn.Do("GET", key))
	if err != nil {
		return err
	}
	if res != p.lock {
		return errors.New("Bad lock")
	}

	_, err = conn.Do("DEL", key)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provisioner) Check(shardID string) (string, error) {
	conn := p.pool.Get()
	defer conn.Close()
	return redis.String(conn.Do("GET", p.redisPrefix+":lock:"+shardID))
}

func (p *Provisioner) Heartbeat(shardID string) error {
	if !p.acquired[shardID] {
		return errors.New("Cannot heartbeat on lock not originally acquired")
	}

	var (
		lastHeartbeat time.Time
		ok            bool
	)

	func() {
		p.heartbeatsMut.RLock()
		defer p.heartbeatsMut.RUnlock()
		lastHeartbeat, ok = p.heartbeats[shardID]
	}()

	if !ok {
		lastHeartbeat = time.Now().Add(-(p.ttl + time.Second))
	}

	now := time.Now()

	if 2*(now.Sub(lastHeartbeat)) < p.ttl {
		return nil
	}

	conn := p.pool.Get()
	defer conn.Close()

	lockKey := p.redisPrefix + ":lock:" + shardID

	res, err := conn.Do("GET", lockKey)
	if err != nil {
		return err
	}

	lock, err := redis.String(res, err)
	if lock == "" {
		return p.TryAcquire(shardID)
	}
	if lock != p.lock {
		return errors.New("Lock changed from " + p.lock + " to " + lock)
	}

	res, err = conn.Do("PEXPIRE", lockKey, int64(p.ttl/time.Millisecond))
	if err != nil {
		err := p.TryAcquire(shardID)
		if err != nil {
			return err
		}
	}

	p.heartbeatsMut.Lock()
	defer p.heartbeatsMut.Unlock()
	p.heartbeats[shardID] = now

	return nil
}

func (p *Provisioner) TTL() time.Duration {
	return p.ttl
}
