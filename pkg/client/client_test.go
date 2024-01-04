package client_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/massdriver-cloud/fogmachine/pkg/client"
	"github.com/massdriver-cloud/fogmachine/pkg/testing/mock"
)

func TestCreate(t *testing.T) {
	cfMock := mock.NewCloudFormationMock()

	cfMock.SetCreateChangeSetReturn(cloudformation.CreateChangeSetOutput{
		Id: aws.String("foo"),
	})

	cfMock.SetDescribeChangeSetReturn(cloudformation.DescribeChangeSetOutput{
		Status: types.ChangeSetStatusCreateComplete,
	})

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("us-west-2"), config.WithAPIOptions([]func(*middleware.Stack) error{cfMock.CloudFormationMiddlewareInjector()}))
	if err != nil {
		t.FailNow()
	}

	mockcf := cloudformation.NewFromConfig(cfg)

	cf, err := client.NewCloudformationClientWithCFClient("bar", 5, 5, mockcf)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	err = cf.CreateChangeset(context.Background(), nil, nil)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	calls := cfMock.GetCallCount()
	if calls["CreateChangeSet"] != 1 || calls["DescribeChangeSet"] != 1 || calls["DescribeStacks"] != 1 {
		t.Logf("failed call counts %v", calls)
		t.Fail()
	}
}
