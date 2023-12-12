package client

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/massdriver-cloud/fogmachine/pkg/eventcache"
	"github.com/rs/zerolog/log"
)

type Client struct {
	client      *cloudformation.Client
	eventCache  *eventcache.EventCache
	stackId     string
	changesetId *string
}

type ChangesetCreator interface {
	CreateChangeset(packageName string, template []byte, parameters []types.Parameter) (*cloudformation.CreateChangeSetOutput, error)
}

type ChangesetExecutor interface {
	ExecuteChangeset(ctx context.Context) error
}

func NewCloudformationClient(packageName string, region string, ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return &Client{
		client:     cloudformation.NewFromConfig(cfg),
		eventCache: eventcache.New(),
		stackId:    packageName,
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
			log.Fatal().Err(err)
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
		log.Fatal().Err(err)
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

	retry := true
	prevStatus := ""

	for retry {
		result, err := c.client.DescribeChangeSet(ctx, params)
		if err != nil {
			return err
		}

		status := string(result.Status)

		if prevStatus == status {
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
		time.Sleep(3 * time.Second)
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

	var channel chan eventcache.Event = make(chan eventcache.Event)

	go c.stackStatusWatcher(channel)
	go c.changeSetExecutionStatusWatcher(channel)

	for msg := range channel {
		log.Info().
			Str("phase", "Execution").
			Str("event_type", msg.Type).
			Str("provisioner_resource_id", msg.ResourceName).
			Str("provider_resource_id", msg.ProviderResourceId).
			Str("status", msg.ResourceStatus).
			Msg("")
	}

	return nil
}

func (c Client) stackStatusWatcher(channel chan eventcache.Event) error {
	retry := true

	for retry {
		params := &cloudformation.DescribeStacksInput{
			StackName: aws.String(c.stackId),
		}

		result, err := c.client.DescribeStacks(context.Background(), params)
		if err != nil {
			close(channel)
			return err
		}

		stack := result.Stacks[0]

		if isTerminalStatus(string(stack.StackStatus)) {
			channel <- eventcache.Event{
				ResourceName:   c.stackId,
				ResourceStatus: string(stack.StackStatus),
				Type:           "Deployment",
			}

			close(channel)
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}

func (c *Client) changeSetExecutionStatusWatcher(channel chan eventcache.Event) error {
	retry := true

	for retry {
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
				channel <- e
			}
		}

		time.Sleep(3 * time.Second)
	}

	return nil
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
