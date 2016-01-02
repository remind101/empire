package redispool

import (
	"net/url"
	"time"

	"github.com/garyburd/redigo/redis"
)

func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// redis://x:passwd@host:port
func NewRedisPool(connstr string) (*redis.Pool, error) {

	u, err := url.Parse(connstr)
	if err != nil {
		return nil, err
	}

	// auth if necessary
	passwd := ""
	if u.User != nil {
		passwd, _ = u.User.Password()
	}

	pool := newPool(u.Host, passwd)

	return pool, nil
}
