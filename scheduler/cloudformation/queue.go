package cloudformation

import (
	"database/sql"
	"fmt"
	"hash/crc32"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

var (
	// This controls how long a pending stack update has to wait in the queue before
	// it gives up.
	lockTimeout = 2 * time.Minute
)

// stackUpdateQueue implements a simple queueing system for performing stack updates.
type stackUpdateQueue struct {
	db *sql.DB

	cloudformationClient
}

// UpdateStack updates a CloudFormation stack with the given input, enqueueing
// it if there is a currently active update. Updates are guaranteed to happen in
// the order that they arrive.
//
// 1. If there are no active updates, the stack will begin updating.
// 2. If there is an active update, this update will be enqueued behind it.
// 3. If there is a pending update, it will be replaced with this update.
func (q *stackUpdateQueue) UpdateStack(input *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	return q.UpdateStackDone(input, make(chan error, 1))
}

func (q *stackUpdateQueue) UpdateStackDone(input *cloudformation.UpdateStackInput, done chan error) (*cloudformation.UpdateStackOutput, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, err
	}

	timeout := int(lockTimeout.Seconds() * 1000)
	_, err = tx.Exec(fmt.Sprintf("SET LOCAL lock_timeout = %d", timeout))
	if err != nil {
		return nil, fmt.Errorf("error setting lock timeout: %v", err)
	}

	key := crc32.ChecksumIEEE([]byte(fmt.Sprintf("stack_%s", *input.StackName)))
	_, err = tx.Exec(`SELECT pg_advisory_lock($1)`, key)
	if err != nil {
		return nil, fmt.Errorf("error obtaining stack update lock: %v", err)
	}

	resp, err := q.updateStack(input)
	if err != nil {
		return resp, err
	}

	// Start up a goroutine that will wait for this stack update to
	// complete, and release the lock when it completes.
	go func(stackName string) {
		defer tx.Commit()

		// Wait for the update to complete.
		// FIXME: Timeout.
		done <- q.cloudformationClient.WaitUntilStackUpdateComplete(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
	}(*input.StackName)

	return resp, nil
}

// updateStack performs a stack update. It returns only when the update is
// complete.
func (q *stackUpdateQueue) updateStack(input *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	stack, err := q.stack(input.StackName)
	if err != nil {
		return nil, err
	}

	// If we're updating a stack, without changing the template, merge in
	// existing parameters with their previous value.
	if input.UsePreviousTemplate != nil && *input.UsePreviousTemplate == true {
		// The parameters that the stack defines. We need to make sure that we
		// provide all parameters in the update (lame).
		definedParams := make(map[string]bool)
		for _, p := range stack.Parameters {
			definedParams[*p.ParameterKey] = true
		}

		// The parameters that are provided in this update.
		providedParams := make(map[string]bool)
		for _, p := range input.Parameters {
			providedParams[*p.ParameterKey] = true
		}

		// Fill in any parameters that weren't provided with their default
		// value.
		for k := range definedParams {
			if !providedParams[k] {
				input.Parameters = append(input.Parameters, &cloudformation.Parameter{
					ParameterKey:     aws.String(k),
					UsePreviousValue: aws.Bool(true),
				})
			}
		}
	}

	resp, err := q.cloudformationClient.UpdateStack(input)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() == "ValidationError" && err.Message() == "No updates are to be performed." {
				return resp, nil
			}
		}

		return resp, fmt.Errorf("error updating stack: %v", err)
	}

	return resp, nil
}

// stack returns the cloudformation.Stack for the given stack name.
func (q *stackUpdateQueue) stack(stackName *string) (*cloudformation.Stack, error) {
	resp, err := q.cloudformationClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})
	if err != nil {
		return nil, err
	}
	return resp.Stacks[0], nil
}
