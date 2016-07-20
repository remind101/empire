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

// InstancePortsResource is a Provisioner that allocates instance ports.
type InstancePortsResource struct {
	ports portAllocator
}

func newInstancePortsProvisioner(resource *InstancePortsResource) *provisioner {
	return &provisioner{
		Create: resource.Create,
		Delete: resource.Delete,
	}
}

func (p *InstancePortsResource) Properties() interface{} {
	return nil
}

func (p *InstancePortsResource) Create(_ context.Context, req customresources.Request) (string, interface{}, error) {
	var port int64
	port, err := p.ports.Get()
	data := map[string]int64{"InstancePort": port}
	id := fmt.Sprintf("%d", port)
	return id, data, err
}

func (p *InstancePortsResource) Delete(_ context.Context, req customresources.Request) error {
	port, err := strconv.Atoi(req.PhysicalResourceId)
	if err != nil {
		return fmt.Errorf("physical resource id should have been a port number: %v", err)
	}
	return p.ports.Put(int64(port))
}
