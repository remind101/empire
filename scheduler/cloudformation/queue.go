package cloudformation

import (
	"database/sql"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

const (
	// Inserts the job into the queue, returning it's job id.
	sqlEnqueue = `INSERT INTO cloudformation_queue (stack) VALUES ($1) RETURNING id`

	// Removes the job from the queue.
	sqlDequeue = `DELETE FROM cloudformation_queue WHERE id = $1`

	// Returns the currently active job for an update.
	sqlActive = `SELECT id FROM cloudformation_queue WHERE stack = $1 ORDER BY id asc LIMIT 1`

	// Checks if this job is invalid (there's another update pending).
	sqlPending = `SELECT count(*) FROM cloudformation_queue WHERE stack = $1 AND id > $2`
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
	var id int
	err := q.db.QueryRow(sqlEnqueue, *input.StackName).Scan(&id)
	if err != nil {
		return nil, err
	}

	// Wait for this job to become active.
	for {
		var pending int
		err := q.db.QueryRow(sqlPending, *input.StackName, id).Scan(&pending)
		if err != nil {
			return nil, err
		}
		if pending > 0 {
			// If there's another pending stack update, we'll treat
			// this one as invalid.
			return nil, nil
		}

		// FIXME: Rate limit.
		var active int
		err = q.db.QueryRow(sqlActive, *input.StackName).Scan(&active)
		if err != nil {
			return nil, err
		}

		if active == id {
			break
		}
	}

	defer q.db.Exec(sqlDequeue, id)
	return q.updateStack(input)
}

// updateStack performs a stack update. It returns only when the update is
// complete.
func (q *stackUpdateQueue) updateStack(input *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	stack, err := q.stack(input.StackName)
	if err != nil {
		return nil, err
	}

	stackName := input.StackName

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

	// Wait for the update to complete.
	// FIXME: Timeout.
	if err := q.cloudformationClient.WaitUntilStackUpdateComplete(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	}); err != nil {
		return resp, err
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
