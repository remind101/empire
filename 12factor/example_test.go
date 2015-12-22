package twelvefactor_test

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/12factor/scheduler/ecs"
)

// Simple example that shows how to run an application with the ECS scheduling
// backend.
func Example() {
	// This is hard coded, but the end goal would be to provide methods for
	// parsing Procfile and docker-compose.yml files into this Manifest
	// type.
	m := twelvefactor.Manifest{
		App: twelvefactor.App{
			ID:    "acme-inc",
			Name:  "acme-inc",
			Image: "remind101/acme-inc:master",
		},
		Processes: []twelvefactor.Process{
			{
				Name:    "web",
				Command: []string{"acme-inc server"},
			},
		},
	}

	// Use ECS as a scheduling backend. Our application will be run as ECS
	// services.
	scheduler := ecs.NewScheduler(session.New())

	// Bring up the application. Creates ECS services as necessary.
	err := scheduler.Up(m)
	if err != nil {
		log.Fatal(err)
	}

	// Scale up our web process.
	err = scheduler.ScaleProcess(m.ID, "web", 2)
	if err != nil {
		log.Fatal(err)
	}

	// Remove the ECS resources.
	err = scheduler.Remove(m.ID)
	if err != nil {
		log.Fatal(err)
	}
}
