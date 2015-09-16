package kinesumeriface

import (
	"time"
)

type Provisioner interface {
	TryAcquire(shardID string) error
	Release(shardID string) error
	Heartbeat(shardID string) error
	TTL() time.Duration
}
