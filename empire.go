package empire

import (
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/deploys"
)

type Empire struct {
	configsService *configs.Service
	deploysService *deploys.Service
}

func (e *Empire) ConfigsService() *configs.Service {
	return e.configsService
}

func (e *Empire) DeploysService() *deploys.Service {
	return e.deploysService
}
