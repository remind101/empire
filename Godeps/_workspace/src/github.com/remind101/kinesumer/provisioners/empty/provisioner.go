package emptyprovisioner

import (
	"time"
)

type Provisioner struct {
}

func (p Provisioner) TryAcquire(shardID string) error {
	return nil
}

func (p Provisioner) Release(shardID string) error {
	return nil
}

func (p Provisioner) Heartbeat(shardID string) error {
	return nil
}

func (p Provisioner) TTL() time.Duration {
	return time.Hour
}
