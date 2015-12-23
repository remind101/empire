// Package ecs provides a scheduler for running 12factor applications using ECS.
package ecs

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/12factor/scheduler/ecs/raw"
	"github.com/remind101/empire/pkg/aws/arn"
)

// ProcessNotFoundError is returned when attempting to operate on a process that
// does not exist.
type ProcessNotFoundError struct {
	Process string
}

// Error implements the error interface.
func (e *ProcessNotFoundError) Error() string {
	return fmt.Sprintf("%s process not found", e.Process)
}

// ecsClient represents a client for interacting with ECS.
type ecsClient interface {
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
	ListTasks(*ecs.ListTasksInput) (*ecs.ListTasksOutput, error)
	DescribeTasks(*ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	StopTask(*ecs.StopTaskInput) (*ecs.StopTaskOutput, error)
}

// Scheduler is an implementation of the twelvefactor.Scheduler interface that
// is backed by ECS.
type Scheduler struct {
	// Cluster is the name of the ECS cluster to operate within. The zero
	// value is the "default" cluster.
	Cluster string

	ecs ecsClient

	// stackBuilder is the StackBuilder that will be used to provision AWS
	// resources.
	stackBuilder StackBuilder
}

// NewScheduler builds a new Scheduler instance backed by an ECS client
// that's configured with the given config.
func NewScheduler(config client.ConfigProvider) *Scheduler {
	return &Scheduler{
		ecs:          ecs.New(config),
		stackBuilder: raw.NewStackBuilder(config),
	}
}

// Up creates or updates the associated ECS services for the individual
// processes within the application and runs them.
func (s *Scheduler) Up(manifest twelvefactor.Manifest) error {
	return s.stackBuilder.Build(manifest)
}

// Remove removes the app and it's associated AWS resources.
func (s *Scheduler) Remove(app string) error {
	return s.stackBuilder.Remove(app)
}

// Restart restarts all ECS services for this application.
func (s *Scheduler) Restart(app string) error {
	return nil
}

// Restart restarts an the ECS service associated with the given process.
func (s *Scheduler) RestartProcess(app string, process string) error {
	// TODO:
	// DescribeService
	// DescribeTaskDefinition
	// RegisterTaskDefinition (Copy)
	// UpdateService
	return nil
}

// StopTask stops an ECS task.
func (s *Scheduler) StopTask(taskID string) error {
	_, err := s.ecs.StopTask(&ecs.StopTaskInput{
		Task:    aws.String(taskID),
		Cluster: aws.String(s.Cluster),
	})
	return err
}

// ScaleProcess scales the associated ECS service for the given app and process
// name.
func (s *Scheduler) ScaleProcess(app, process string, desired int) error {
	service, err := s.Service(app, process)
	if err != nil {
		return err
	}

	_, err = s.ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      aws.String(s.Cluster),
		DesiredCount: aws.Int64(int64(desired)),
		Service:      aws.String(service),
	})
	return err
}

// Service returns the name of the ECS service for the given process.
func (s *Scheduler) Service(app, process string) (string, error) {
	services, err := s.stackBuilder.Services(app)
	if err != nil {
		return "", err
	}

	// If there's no matching ECS service for this process, return an error.
	service, ok := services[process]
	if !ok {
		return service, &ProcessNotFoundError{Process: process}
	}

	return service, nil
}

// Tasks returns the RUNNING and PENDING ECS tasks for the ECS services.
func (s *Scheduler) Tasks(app string) ([]twelvefactor.Task, error) {
	services, err := s.stackBuilder.Services(app)
	if err != nil {
		return nil, err
	}

	var tasks []twelvefactor.Task
	for _, service := range services {
		serviceTasks, err := s.ServiceTasks(service)
		if err != nil {
			return tasks, err
		}
		tasks = append(tasks, serviceTasks...)
	}

	return tasks, nil
}

// ServiceTasks returns the Tasks running for the given ECS service.
func (s *Scheduler) ServiceTasks(service string) ([]twelvefactor.Task, error) {
	listResp, err := s.ecs.ListTasks(&ecs.ListTasksInput{
		Cluster:     aws.String(s.Cluster),
		ServiceName: aws.String(service),
	})
	if err != nil {
		return nil, err
	}

	// No tasks.
	if len(listResp.TaskArns) == 0 {
		return nil, nil
	}

	describeResp, err := s.ecs.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(s.Cluster),
		Tasks:   listResp.TaskArns,
	})
	if err != nil {
		return nil, err
	}

	var tasks []twelvefactor.Task
	for _, task := range describeResp.Tasks {
		id, err := arn.ResourceID(*task.TaskArn)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, twelvefactor.Task{
			ID:    id,
			State: *task.LastStatus,
		})
	}

	return tasks, nil
}
