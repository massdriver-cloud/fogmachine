package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/massdriver-cloud/fogmachine/pkg/eventcache"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

//go:generate go run ../../generate/main.go

type Client struct {
	client       *cloudformation.Client
	eventCache   *eventcache.EventCache
	stackID      string
	changesetID  *string
	pollIntervel time.Duration
	timeout      time.Duration
}

func NewCloudformationClient(ctx context.Context, packageName, region string, t, pollInterval int) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return NewCloudformationClientWithCFClient(packageName, t, pollInterval, cloudformation.NewFromConfig(cfg))
}

func NewCloudformationClientWithCFClient(packageName string, t, pollInterval int, cfClient *cloudformation.Client) (*Client, error) {
	return &Client{
		client:       cfClient,
		eventCache:   eventcache.New(),
		stackID:      packageName,
		pollIntervel: time.Duration(pollInterval) * time.Second,
		timeout:      time.Duration(t) * time.Second,
	}, nil
}

func (c *Client) CreateChangeset(ctx context.Context, template []byte, parameters []types.Parameter) error {
	input := &cloudformation.CreateChangeSetInput{
		ChangeSetName: aws.String(fmt.Sprintf("%s-%d", c.stackID, time.Now().Unix())),
		ChangeSetType: types.ChangeSetTypeCreate,
		StackName:     aws.String(c.stackID),
		Description:   aws.String("Changeset created via Fog-Machine"),
		TemplateBody:  aws.String(string(template)),
		Parameters:    parameters,
	}

	ok, err := c.stackExists(ctx)
	if err != nil {
		return err
	}

	if ok {
		input.ChangeSetType = types.ChangeSetTypeUpdate
		err = c.primeEventCache(ctx)
		if err != nil {
			return err
		}
	}

	log.Info().Str("phase", "Changeset").Msg("Creating changeset")

	response, err := c.client.CreateChangeSet(ctx, input)
	if err != nil {
		return err
	}

	c.changesetID = response.Id

	return c.changeSetStatusWatcher(ctx)
}

func (c *Client) primeEventCache(ctx context.Context) error {
	params := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(c.stackID),
	}

	result, err := c.client.DescribeStackEvents(ctx, params)
	if err != nil {
		return err
	}

	for _, event := range result.StackEvents {
		c.eventCache.AddEvent(*event.EventId, c.eventCache.EventFromStack(event, "Cache"))
	}

	return nil
}

func (c Client) stackExists(ctx context.Context) (bool, error) {
	params := cloudformation.DescribeStacksInput{
		StackName: aws.String(c.stackID),
	}

	response, err := c.client.DescribeStacks(ctx, &params)
	if err != nil {
		if !errorIsDoesNotExist(err) {
			return false, err
		}
		return false, nil
	}

	if len(response.Stacks) != 1 {
		return false, nil
	}

	inReview := response.Stacks[0].StackStatus != "REVIEW_IN_PROGRESS"

	return inReview, nil
}

func (c Client) changeSetStatusWatcher(ctx context.Context) error {
	params := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: c.changesetID,
		StackName:     aws.String(c.stackID),
	}

	start := time.Now()
	var prevStatus string

	for {
		result, err := c.client.DescribeChangeSet(ctx, params)
		if err != nil {
			return err
		}

		status := string(result.Status)

		if prevStatus == status {
			time.Sleep(c.pollIntervel)
			continue
		}

		if isTerminalStatus(status) {
			var message string

			if result.StatusReason != nil {
				message = *result.StatusReason
			}

			log.Info().
				Str("changesetId", *c.changesetID).
				Str("stackName", c.stackID).
				Str("status", status).
				Str("phase", "Changeset").
				Msg(message)
			return nil
		}

		log.Info().
			Str("Phase", "Changeset").
			Str("ChangesetId", *c.changesetID).
			Str("StackName", c.stackID).
			Str("Status", status).
			Msg("")

		prevStatus = status

		if time.Since(start) > c.timeout {
			break
		}

		time.Sleep(c.pollIntervel)
	}

	return errors.New("changeset failed to reach a terminal state")
}

func (c Client) ExecuteChangeSet(ctx context.Context) error {
	log.Info().Str("phase", "Execution").Msg("Validating changeset")
	params := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: c.changesetID,
		StackName:     aws.String(c.stackID),
	}

	result, err := c.client.DescribeChangeSet(ctx, params)
	if err != nil {
		return err
	}

	if len(result.Changes) == 0 {
		log.Info().Str("phase", "Execution").Msg("No changes in changeset")
		return nil
	}

	log.Info().Str("phase", "Execution").Msg("Executing changeset")

	input := &cloudformation.ExecuteChangeSetInput{
		StackName:     aws.String(c.stackID),
		ChangeSetName: c.changesetID,
	}

	_, err = c.client.ExecuteChangeSet(ctx, input)
	if err != nil {
		return err
	}

	return c.runWatchers(ctx)
}

func (c Client) ExecuteDestroyStack(ctx context.Context) error {
	log.Info().Str("phase", "Execution").Msg("Verifying stack exists")

	if ok, err := c.stackExists(ctx); err != nil {
		return err
	} else if !ok {
		log.Info().Str("phase", "Execution").Msg("Stack does not exist, nothing to destroy")
		return nil
	}

	log.Debug().Str("phase", "Execution").Msg("Priming cache")

	err := c.primeEventCache(ctx)
	if err != nil {
		return err
	}

	log.Info().Str("phase", "Execution").Msg("Destroying stack")

	input := &cloudformation.DeleteStackInput{
		StackName: aws.String(c.stackID),
	}

	_, err = c.client.DeleteStack(ctx, input)
	if err != nil {
		return err
	}

	if err = c.runWatchers(ctx); err != nil {
		// On destroy we will hit this error so we know the stack is gone, anything else should return
		if !errorIsDoesNotExist(err) {
			return err
		}
		log.Info().Str("phase", "Execution").Msg("Stack destroyed successfully")
	}

	return nil
}

func (c Client) runWatchers(ctx context.Context) error {
	ctx, cancel := context.WithTimeoutCause(ctx, c.timeout, errors.New("reached timout"))

	var errGroup errgroup.Group

	errGroup.Go(func() error { return c.stackStatusWatcher(ctx, cancel) })
	errGroup.Go(func() error { return c.changeSetExecutionStatusWatcher(ctx, cancel) })

	if err := errGroup.Wait(); err != nil {
		return err
	}

	if ctx.Err() != nil {
		// We don't care if the context was canceled since that's how we signal the go routines
		// so only log if the deadline was exceeded
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Info().Msg("Reached timeout deadline waiting for stack to complete")
		}
	}

	return nil
}

func (c Client) stackStatusWatcher(ctx context.Context, cancel context.CancelFunc) error {
	defer cancel()

	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(c.stackID),
	}

	for {
		result, err := c.client.DescribeStacks(ctx, params)
		if err != nil {
			return err
		}

		stack := result.Stacks[0]

		if isTerminalStatus(string(stack.StackStatus)) {
			log.Info().
				Str("phase", "Execution").
				Str("event_type", "Deployment").
				Str("provisioner_resource_id", c.stackID).
				Str("provider_resource_id", "").
				Str("status", string(stack.StackStatus)).
				Msg("")

			return nil
		}

		// Sleep before the context check so if it was canceled we exit without trying the API again
		time.Sleep(c.pollIntervel)

		select {
		case <-ctx.Done():
			log.Debug().Str("phase", "Execution").Str("func", "stack").Msg("ctx canceled")
			return nil
		default:
			log.Debug().Str("phase", "Execution").Msg("stack Going to poll again")
		}
	}
}

func (c *Client) changeSetExecutionStatusWatcher(ctx context.Context, cancel context.CancelFunc) error {
	defer cancel()

	params := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(c.stackID),
	}

	for {
		result, err := c.client.DescribeStackEvents(ctx, params)
		if err != nil {
			return err
		}

		for i := len(result.StackEvents) - 1; i >= 0; i-- {
			event := result.StackEvents[i]

			if !c.eventCache.EventExists(*event.EventId) {
				e := c.eventCache.EventFromStack(event, "Resource")
				c.eventCache.AddEvent(*event.EventId, e)
				log.Info().
					Str("phase", "Execution").
					Str("event_type", e.Type).
					Str("provisioner_resource_id", e.ResourceName).
					Str("provider_resource_id", e.ProviderResourceID).
					Str("status", e.ResourceStatus).
					Msg("")
			}
		}

		// Sleep before the context check so if it was canceled we exit without trying the API again
		// This also stops any logs coming out if the stack status log already happened
		time.Sleep(c.pollIntervel)

		select {
		case <-ctx.Done():
			log.Debug().Str("phase", "Execution").Str("func", "changeSet").Msg("ctx canceled")
			return nil
		default:
			log.Debug().Str("phase", "Execution").Msg("changeSet Going to poll again")
		}
	}
}

func isTerminalStatus(status string) bool {
	switch status {
	case string(types.ChangeSetStatusFailed):
		return true
	case string(types.ChangeSetStatusCreateComplete):
		return true
	case string(types.StackStatusUpdateComplete):
		return true
	case string(types.StackStatusUpdateFailed):
		return true
	case string(types.StackStatusCreateFailed):
		return true
	default:
		return false
	}
}

func errorIsDoesNotExist(err error) bool {
	return strings.Contains(err.Error(), "does not exist")
}
