package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/deploys"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

type Empire struct {
	appsService     *apps.Service
	configsService  *configs.Service
	deploysService  *deploys.Service
	releasesService *releases.Service
	slugsService    *slugs.Service
}

func New() *Empire {
	return &Empire{}
}

func (e *Empire) AppsService() *apps.Service {
	if e.appsService == nil {
		e.appsService = apps.NewService(nil)
	}

	return e.appsService
}

func (e *Empire) ConfigsService() *configs.Service {
	if e.configsService == nil {
		e.configsService = configs.NewService(nil)
	}

	return e.configsService
}

func (e *Empire) DeploysService() *deploys.Service {
	if e.deploysService == nil {
		e.deploysService = &deploys.Service{
			AppsService:     e.AppsService(),
			ConfigsService:  e.ConfigsService(),
			SlugsService:    e.SlugsService(),
			ReleasesService: e.ReleasesService(),
		}
	}

	return e.deploysService
}

func (e *Empire) ReleasesService() *releases.Service {
	if e.releasesService == nil {
		e.releasesService = releases.NewService(nil)
	}

	return e.releasesService
}

func (e *Empire) SlugsService() *slugs.Service {
	if e.slugsService == nil {
		e.slugsService = slugs.NewService(nil, nil)
	}

	return e.slugsService
}
