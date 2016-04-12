// Package ecs provides a scheduler for running 12factor applications using ECS.
package ecs

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/pkg/aws/arn"
	"github.com/remind101/empire/pkg/bytesize"
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
	DescribeServices(*ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error)
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
	ListTasks(*ecs.ListTasksInput) (*ecs.ListTasksOutput, error)
	DescribeTasks(*ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	StopTask(*ecs.StopTaskInput) (*ecs.StopTaskOutput, error)
	DescribeTaskDefinition(*ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error)
	RegisterTaskDefinition(*ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
}

// Scheduler is an implementation of the twelvefactor.Scheduler interface that
// is backed by ECS.
type Scheduler struct {
	// Cluster is the name of the ECS cluster to operate within. The zero
	// value is the "default" cluster.
	Cluster string

	// StackBuilder is the StackBuilder that will be used to provision AWS
	// resources.
	StackBuilder StackBuilder

	ecs ecsClient
}

// NewScheduler builds a new Scheduler instance backed by an ECS client
// that's configured with the given config.
func NewScheduler(config client.ConfigProvider) *Scheduler {
	return &Scheduler{
		ecs: ecs.New(config),
	}
}

// Up creates or updates the associated ECS services for the individual
// processes within the application and runs them.
func (s *Scheduler) Up(manifest twelvefactor.Manifest) error {
	return s.StackBuilder.Build(manifest)
}

// Remove removes the app and it's associated AWS resources.
func (s *Scheduler) Remove(app string) error {
	return s.StackBuilder.Remove(app)
}

// Restart restarts all ECS services for this application.
func (s *Scheduler) Restart(app string) error {
	services, err := s.StackBuilder.Services(app)
	if err != nil {
		return err
	}

	for _, service := range services {
		// TODO(ejholmes): Parallelize
		if err := s.RestartService(service); err != nil {
			return err
		}
	}

	return nil
}

// Restart restarts an the ECS service associated with the given process.
func (s *Scheduler) RestartProcess(app string, process string) error {
	service, err := s.Service(app, process)
	if err != nil {
		return err
	}

	return s.RestartService(service)
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

// Tasks returns the RUNNING and PENDING ECS tasks for the ECS services.
func (s *Scheduler) Tasks(app string) ([]twelvefactor.Task, error) {
	services, err := s.StackBuilder.Services(app)
	if err != nil {
		return nil, err
	}

	var tasks []twelvefactor.Task
	for _, service := range services {
		// TODO(ejholmes): Parallelize this.
		serviceTasks, err := s.ServiceTasks(service)
		if err != nil {
			return tasks, err
		}
		tasks = append(tasks, serviceTasks...)
	}

	return tasks, nil
}

// RestartService "restarts" the given service. We fake a restart by creating a
// new task definition from the current one, and updating the service with it.
func (s *Scheduler) RestartService(service string) error {
	taskDefinition, err := s.CopyTaskDefinition(service)
	if err != nil {
		return err
	}

	_, err = s.ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:        aws.String(s.Cluster),
		TaskDefinition: taskDefinition.TaskDefinitionArn,
		Service:        aws.String(service),
	})
	return err
}

// CopyTaskDefinition copies a task definition for an ECS service, returning the
// new task definition.
func (s *Scheduler) CopyTaskDefinition(service string) (*ecs.TaskDefinition, error) {
	taskDefinition, err := s.TaskDefinition(service)
	if err != nil {
		return nil, err
	}

	resp, err := s.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: taskDefinition.ContainerDefinitions,
		Family:               taskDefinition.Family,
		Volumes:              taskDefinition.Volumes,
	})
	if err != nil {
		return nil, err
	}

	return resp.TaskDefinition, nil
}

// TaskDefinition returns the task definition for an ECS service.
func (s *Scheduler) TaskDefinition(service string) (*ecs.TaskDefinition, error) {
	serviceResp, err := s.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(s.Cluster),
		Services: []*string{aws.String(service)},
	})
	if err != nil {
		return nil, err
	}

	// TODO(ejholmes): Handle unexpected length.
	taskDefinition := serviceResp.Services[0].TaskDefinition
	descResp, err := s.ecs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: taskDefinition,
	})
	if err != nil {
		return nil, err
	}

	// TODO(ejholmes): Handle Failures
	return descResp.TaskDefinition, nil
}

// Service returns the name of the ECS service for the given process.
func (s *Scheduler) Service(app, process string) (string, error) {
	services, err := s.StackBuilder.Services(app)
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

	taskDefinition, err := s.TaskDefinition(service)
	if err != nil {
		return nil, err
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

		state := *task.LastStatus
		var updatedAt time.Time
		switch state {
		case "PENDING":
			updatedAt = *task.CreatedAt
		case "RUNNING":
			updatedAt = *task.StartedAt
		case "STOPPED":
			updatedAt = *task.StoppedAt
		}

		containerDefinition := *taskDefinition.ContainerDefinitions[0]

		var command []string
		for _, s := range containerDefinition.Command {
			command = append(command, *s)
		}

		tasks = append(tasks, twelvefactor.Task{
			ID:        id,
			Version:   "TODO",
			State:     state,
			UpdatedAt: updatedAt,
			Process:   *containerDefinition.Name,
			Command:   command,
			Memory:    int(*containerDefinition.Memory) * int(bytesize.MB),
			CPUShares: int(*containerDefinition.Cpu),
		})
	}

	return tasks, nil
}
