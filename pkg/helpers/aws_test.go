package helpers_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/pkg/helpers"
	mockaws "github.com/convox/rack/pkg/mock/aws"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCloudformationDescribe(t *testing.T) {
	cf := &mockaws.CloudFormationAPI{}
	expectStack := &cloudformation.Stack{
		StackId:   aws.String("id1"),
		StackName: aws.String("stack1"),
	}

	cf.On("DescribeStacks", mock.Anything).Return(func(st *cloudformation.DescribeStacksInput) *cloudformation.DescribeStacksOutput {
		require.Equal(t, expectStack.StackName, st.StackName)
		return &cloudformation.DescribeStacksOutput{
			Stacks: []*cloudformation.Stack{
				expectStack,
			},
		}
	}, nil)

	stack, err := helpers.CloudformationDescribe(cf, *expectStack.StackName)
	require.NoError(t, err)
	require.Equal(t, expectStack, stack)
}

func TestCloudformationInstall(t *testing.T) {
	cf := &mockaws.CloudFormationAPI{}
	expectStack := &cloudformation.Stack{
		StackId:     aws.String("id1"),
		StackName:   aws.String("stack1"),
		StackStatus: aws.String("CREATE_COMPLETE"),
	}

	cf.On("CreateChangeSet", mock.Anything).Return(func(st *cloudformation.CreateChangeSetInput) *cloudformation.CreateChangeSetOutput {
		return &cloudformation.CreateChangeSetOutput{
			StackId: expectStack.StackId,
		}
	}, nil)

	cf.On("WaitUntilChangeSetCreateComplete", mock.Anything).Return(func(in *cloudformation.DescribeChangeSetInput) error {
		require.Equal(t, expectStack.StackId, in.StackName)
		return nil
	})

	cf.On("DescribeChangeSet", mock.Anything).Return(func(in *cloudformation.DescribeChangeSetInput) *cloudformation.DescribeChangeSetOutput {
		require.Equal(t, expectStack.StackId, in.StackName)
		return &cloudformation.DescribeChangeSetOutput{}
	}, nil)

	cf.On("ExecuteChangeSet", mock.Anything).Return(nil, func(in *cloudformation.ExecuteChangeSetInput) error {
		require.Equal(t, expectStack.StackId, in.StackName)
		return nil
	})

	cf.On("DescribeStacks", mock.Anything).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			expectStack,
		},
	}, nil)

	err := helpers.CloudformationInstall(cf, *expectStack.StackName, "", map[string]string{}, map[string]string{}, func(i1, i2 int) {})
	require.NoError(t, err)
}

func TestCloudformationParameters(t *testing.T) {
	data := `
Parameters:
  key1: 1
  key2: 2
`
	expect := map[string]bool{
		"key1": true,
		"key2": true,
	}
	got, err := helpers.CloudformationParameters([]byte(data))
	require.NoError(t, err)
	require.Equal(t, expect, got)
}

func TestCloudformationUninstall(t *testing.T) {
	cf := &mockaws.CloudFormationAPI{}
	expectStack := &cloudformation.Stack{
		StackId:     aws.String("id1"),
		StackName:   aws.String("stack1"),
		StackStatus: aws.String("CREATE_COMPLETE"),
	}

	cf.On("DeleteStack", mock.Anything).Return(nil, nil)

	cf.On("WaitUntilStackDeleteComplete", mock.Anything).Return(func(in *cloudformation.DescribeStacksInput) error {
		require.Equal(t, expectStack.StackName, in.StackName)
		return nil
	})

	err := helpers.CloudformationUninstall(cf, *expectStack.StackName)
	require.NoError(t, err)
}

func TestCloudformationUpdate(t *testing.T) {
	cf := &mockaws.CloudFormationAPI{}
	expectStack := &cloudformation.Stack{
		StackId:   aws.String("id1"),
		StackName: aws.String("stack1"),
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("param1"),
				ParameterValue: aws.String("old"),
			},
		},
	}

	cf.On("DescribeStacks", mock.Anything).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			expectStack,
		},
	}, nil)

	cf.On("UpdateStack", mock.Anything).Return(func(in *cloudformation.UpdateStackInput) *cloudformation.UpdateStackOutput {
		require.Equal(t, expectStack.Parameters[0].ParameterKey, in.Parameters[0].ParameterKey)
		require.Equal(t, aws.String("new"), in.Parameters[0].ParameterValue)
		return nil
	}, nil)

	err := helpers.CloudformationUpdate(cf, *expectStack.StackName, "", map[string]string{
		"param1": "new",
	}, map[string]string{}, "")
	require.NoError(t, err)
}

func TestCloudWatchLogsStream(t *testing.T) {
	var calls []time.Time

	cf := &mockaws.CloudWatchLogsAPI{}
	cf.On("FilterLogEvents", mock.Anything).Return(func(in *cloudwatchlogs.FilterLogEventsInput) *cloudwatchlogs.FilterLogEventsOutput {
		calls = append(calls, time.Now())
		return &cloudwatchlogs.FilterLogEventsOutput{
			Events: []*cloudwatchlogs.FilteredLogEvent{
				{
					EventId:       aws.String("id1"),
					IngestionTime: aws.Int64(time.Now().Unix()),
					LogStreamName: aws.String("strm"),
					Message:       aws.String("hello"),
					Timestamp:     aws.Int64(time.Now().Unix()),
				},
			},
		}
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(1500 * time.Millisecond)
		cancel()
	}()

	wc := &myWriteCloser{
		bufio.NewWriter(&bytes.Buffer{}),
	}
	err := helpers.CloudWatchLogsStream(ctx, cf, wc, "grp", "strm", structs.LogsOptions{})
	require.NoError(t, err)
	cf.AssertNumberOfCalls(t, "FilterLogEvents", 2)
	require.Len(t, calls, 2)
	require.GreaterOrEqual(t, calls[1].Sub(calls[0]), time.Second)
}

func TestCloudWatchLogsStreamNoSince(t *testing.T) {
	now := time.Date(2025, 4, 9, 22, 0, 0, 0, time.UTC)
	oldNow := helpers.TimeNow
	helpers.TimeNow = func() time.Time { return now }
	defer func() { helpers.TimeNow = oldNow }()

	var starts []int64

	cw := &mockaws.CloudWatchLogsAPI{}
	cw.On("FilterLogEvents", mock.Anything).Return(func(in *cloudwatchlogs.FilterLogEventsInput) (*cloudwatchlogs.FilterLogEventsOutput, error) {
		starts = append(starts, aws.Int64Value(in.StartTime))
		if len(starts) == 1 {
			return &cloudwatchlogs.FilterLogEventsOutput{}, nil
		}
		return nil, errors.New("stop")
	})

	wc := &myWriteCloser{
		bufio.NewWriter(&bytes.Buffer{}),
	}
	err := helpers.CloudWatchLogsStream(context.Background(), cw, wc, "grp", "strm", structs.LogsOptions{})
	require.EqualError(t, err, "stop")

	want := now.Add(-1 * time.Hour).UnixMilli()
	require.Equal(t, []int64{want, want}, starts)
}

type myWriteCloser struct {
	*bufio.Writer
}

func (mwc *myWriteCloser) Close() error {
	// Noop
	return nil
}
