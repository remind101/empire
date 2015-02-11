package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/deploys"
)

type Empire struct {
	appsService    *apps.Service
	configsService *configs.Service
	deploysService *deploys.Service
}

func New() *Empire {
	return &Empire{}
}

func (e *Empire) AppsService() *apps.Service {
	return e.appsService
}

func (e *Empire) ConfigsService() *configs.Service {
	return e.configsService
}

func (e *Empire) DeploysService() *deploys.Service {
	return e.deploysService
}
