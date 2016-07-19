package cloudformation

import (
	"fmt"
	"strconv"

	"github.com/remind101/empire/pkg/cloudformation/customresources"

	"golang.org/x/net/context"
)

type portAllocator interface {
	Get() (int64, error)
	Put(port int64) error
}

// InstancePortsProvisioner is a Provisioner that allocates instance ports.
type InstancePortsProvisioner struct {
	ports portAllocator
}

func (p *InstancePortsProvisioner) Properties() interface{} {
	return nil
}

func (p *InstancePortsProvisioner) Provision(_ context.Context, req customresources.Request) (id string, data interface{}, err error) {
	switch req.RequestType {
	case customresources.Create:
		var port int64
		port, err = p.ports.Get()
		if err != nil {
			return
		}
		id = fmt.Sprintf("%d", port)
		data = map[string]int64{"InstancePort": port}
	case customresources.Delete:
		port, err2 := strconv.Atoi(req.PhysicalResourceId)
		if err2 != nil {
			err = fmt.Errorf("physical resource id should have been a port number: %v", err2)
			return
		}
		id = req.PhysicalResourceId
		err = p.ports.Put(int64(port))
	default:
		err = fmt.Errorf("%s is not supported", req.RequestType)
	}

	return
}
