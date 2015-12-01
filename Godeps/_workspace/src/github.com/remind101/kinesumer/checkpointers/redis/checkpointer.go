package redischeckpointer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	k "github.com/remind101/kinesumer/interface"
)

type Checkpointer struct {
	heads       map[string]string
	c           chan k.Record
	mut         sync.Mutex
	pool        *redis.Pool
	redisPrefix string
	savePeriod  time.Duration
	wg          sync.WaitGroup
	modified    bool
	errHandler  func(k.Error)
	readOnly    bool
}

type Options struct {
	ReadOnly    bool
	SavePeriod  time.Duration
	RedisPool   *redis.Pool
	RedisPrefix string
	ErrHandler  func(k.Error)
}

type Error struct {
	origin   error
	severity string
}

func (e *Error) Severity() string { return e.severity }

func (e *Error) Origin() error { return e.origin }

func (e *Error) Error() string { return e.origin.Error() }

func New(opt *Options) (*Checkpointer, error) {
	save := opt.SavePeriod
	if save == 0 {
		save = 5 * time.Second
	}

	if opt.ErrHandler == nil {
		opt.ErrHandler = func(err k.Error) {
			panic(err)
		}
	}

	return &Checkpointer{
		heads:       make(map[string]string),
		c:           make(chan k.Record),
		mut:         sync.Mutex{},
		pool:        opt.RedisPool,
		redisPrefix: opt.RedisPrefix,
		savePeriod:  save,
		modified:    true,
		errHandler:  opt.ErrHandler,
		readOnly:    opt.ReadOnly,
	}, nil
}

func (r *Checkpointer) DoneC() chan<- k.Record {
	return r.c
}

func (r *Checkpointer) Sync() {
	if r.readOnly {
		return
	}

	r.mut.Lock()
	defer r.mut.Unlock()
	if len(r.heads) > 0 && r.modified {
		conn := r.pool.Get()
		defer conn.Close()
		if _, err := conn.Do("HMSET", redis.Args{r.redisPrefix + ".sequence"}.AddFlat(r.heads)...); err != nil {
			r.errHandler(&Error{err, k.EWarn})
		}
		r.modified = false
	}
}

func (r *Checkpointer) RunCheckpointer() {
	defer func() {
		if val := recover(); val != nil {
			err := errors.New(fmt.Sprintf("%v", val))
			r.errHandler(&Error{err, k.ECrit})
		}
	}()
	saveTicker := time.NewTicker(r.savePeriod).C
loop:
	for {
		select {
		case <-saveTicker:
			r.Sync()
		case state, ok := <-r.c:
			if !ok {
				break loop
			}
			r.mut.Lock()
			r.heads[state.ShardId()] = state.SequenceNumber()
			r.modified = true
			r.mut.Unlock()
		}
	}
	r.Sync()
	r.wg.Done()
}

func (r *Checkpointer) Begin() error {
	r.wg.Add(1)
	go r.RunCheckpointer()
	return nil
}

func (r *Checkpointer) End() {
	close(r.c)
	r.wg.Wait()
}

func (r *Checkpointer) GetStartSequence(shardID string) string {
	conn := r.pool.Get()
	defer conn.Close()

	var seq string
	res, err := conn.Do("HGET", r.redisPrefix+".sequence", shardID)
	seq, err = redis.String(res, err)

	if err == nil {
		return seq
	} else {
		return ""
	}
}
