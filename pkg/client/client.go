package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/massdriver-cloud/fogmachine/pkg/eventcache"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Client struct {
	client       *cloudformation.Client
	eventCache   *eventcache.EventCache
	stackId      string
	changesetId  *string
	pollIntervel time.Duration
	timeout      time.Duration
}

type ChangesetCreator interface {
	CreateChangeset(packageName string, template []byte, parameters []types.Parameter) (*cloudformation.CreateChangeSetOutput, error)
}

type ChangesetExecutor interface {
	ExecuteChangeset(ctx context.Context) error
}

func NewCloudformationClient(ctx context.Context, packageName string, region string, t, pollInterval int) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return &Client{
		client:       cloudformation.NewFromConfig(cfg),
		eventCache:   eventcache.New(),
		stackId:      packageName,
		pollIntervel: time.Duration(pollInterval) * time.Second,
		timeout:      time.Duration(t) * time.Second,
	}, nil
}

func (c *Client) CreateChangeset(template []byte, parameters []types.Parameter, ctx context.Context) error {
	input := &cloudformation.CreateChangeSetInput{
		ChangeSetName: aws.String(fmt.Sprintf("%s-%d", c.stackId, time.Now().Unix())),
		StackName:     aws.String(c.stackId),
		Description:   aws.String("Changeset created via Fog-Machine"),
		TemplateBody:  aws.String(string(template)),
		Parameters:    parameters,
	}

	if c.stackExists(ctx) {
		input.ChangeSetType = types.ChangeSetTypeUpdate
		err := c.primeEventCache()
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
	} else {
		input.ChangeSetType = types.ChangeSetTypeCreate
	}

	log.Info().Str("phase", "Changeset").Msg("Creating changeset")

	response, err := c.client.CreateChangeSet(ctx, input)
	if err != nil {
		return err
	}

	c.changesetId = response.Id

	return c.changeSetStatusWatcher(ctx)
}

func (c *Client) primeEventCache() error {
	params := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(c.stackId),
	}

	result, err := c.client.DescribeStackEvents(context.Background(), params)
	if err != nil {
		return err
	}

	for _, event := range result.StackEvents {
		c.eventCache.AddEvent(*event.EventId, c.eventCache.EventFromStack(event, "Cache"))
	}

	return nil
}

/*
TODO:
- parse stack doesnt exist errors.
*/

func (c Client) stackExists(ctx context.Context) bool {
	params := cloudformation.DescribeStacksInput{
		StackName: aws.String(c.stackId),
	}

	response, err := c.client.DescribeStacks(ctx, &params)
	if err != nil {
		return false
	}

	if len(response.Stacks) != 1 {
		return false
	}

	inReview := response.Stacks[0].StackStatus != "REVIEW_IN_PROGRESS"

	return inReview
}

func (c Client) changeSetStatusWatcher(ctx context.Context) error {
	params := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: c.changesetId,
		StackName:     aws.String(c.stackId),
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
			message := ""

			if result.StatusReason != nil {
				message = string(*result.StatusReason)
			}

			log.Info().
				Str("changesetId", *c.changesetId).
				Str("stackName", c.stackId).
				Str("status", status).
				Str("phase", "Changeset").
				Msg(message)
			return nil
		}

		log.Info().
			Str("Phase", "Changeset").
			Str("ChangesetId", *c.changesetId).
			Str("StackName", c.stackId).
			Str("Status", status).
			Msg("")

		prevStatus = status

		if time.Since(start) > c.timeout {
			break
		}

		time.Sleep(c.pollIntervel)
	}

	return fmt.Errorf("Changeset failed to reach a terminal state")
}

func (c Client) ExecuteChangeSet(ctx context.Context) error {
	log.Info().Str("phase", "Execution").Msg("Executing changeset")
	input := &cloudformation.ExecuteChangeSetInput{
		StackName:     aws.String(c.stackId),
		ChangeSetName: c.changesetId,
	}

	_, err := c.client.ExecuteChangeSet(ctx, input)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeoutCause(ctx, c.timeout, errors.New("reached timout"))

	var errGroup errgroup.Group

	errGroup.Go(func() error { return c.stackStatusWatcher(ctx, cancel) })
	errGroup.Go(func() error { return c.changeSetExecutionStatusWatcher(ctx, cancel) })

	if err := errGroup.Wait(); err != nil {
		log.Fatal().Err(err).Msg("")
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

	for {
		params := &cloudformation.DescribeStacksInput{
			StackName: aws.String(c.stackId),
		}

		result, err := c.client.DescribeStacks(context.Background(), params)
		if err != nil {
			return err
		}

		stack := result.Stacks[0]

		if isTerminalStatus(string(stack.StackStatus)) {
			log.Info().
				Str("phase", "Execution").
				Str("event_type", "Deployment").
				Str("provisioner_resource_id", c.stackId).
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

	for {
		params := &cloudformation.DescribeStackEventsInput{
			StackName: aws.String(c.stackId),
		}

		result, err := c.client.DescribeStackEvents(context.Background(), params)
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
					Str("provider_resource_id", e.ProviderResourceId).
					Str("status", string(e.ResourceStatus)).
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
