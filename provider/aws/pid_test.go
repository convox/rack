package aws_test

import (
	"testing"

	amazon "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
)

func makeTestPID() *aws.PID {
	return &aws.PID{
		TaskID:      "444444444444",
		ContainerID: "555555555555",
	}
}

func makeFakeECSTask() *ecs.Task {
	return &ecs.Task{
		TaskArn: amazon.String("123e4567-e89b-12d3-a456-444444444444"),
		Containers: []*ecs.Container{
			&ecs.Container{
				Name:         amazon.String("foo"),
				ContainerArn: amazon.String("123e4567-e89b-12d3-a456-555555555555"),
			},
			&ecs.Container{
				Name:         amazon.String("bar"),
				ContainerArn: amazon.String("123e4567-e89b-12d3-a456-777777777777"),
			},
		},
	}
}

func TestParsePIDSucceeds(t *testing.T) {
	pid, err := aws.ParsePID("444444444444-555555555555")

	assert.NoError(t, err)
	assert.Equal(t, "444444444444", pid.TaskID)
	assert.Equal(t, "555555555555", pid.ContainerID)
}

func TestParsePIDFailsTooLong(t *testing.T) {
	_, err := aws.ParsePID("charlie-delta-whiskey-tango")
	assert.Error(t, err)

	_, err = aws.ParsePID("oscarkilo")
	assert.Error(t, err)
}

func TestPIDFromArns(t *testing.T) {
	pid := aws.PIDFromArns("123e4567-e89b-12d3-a456-333333333333", "123e4567-e89b-12d3-a456-444444444444")

	assert.Equal(t, "333333333333", pid.TaskID)
	assert.Equal(t, "444444444444", pid.ContainerID)
}

func TestPIDFromTask(t *testing.T) {
	task := makeFakeECSTask()

	pid1, err := aws.PIDFromTask(task, "foo")

	assert.NoError(t, err)
	assert.Equal(t, "444444444444", pid1.TaskID)
	assert.Equal(t, "555555555555", pid1.ContainerID)

	pid2, err := aws.PIDFromTask(task, "bar")
	assert.NoError(t, err)
	assert.Equal(t, "444444444444", pid2.TaskID)
	assert.Equal(t, "777777777777", pid2.ContainerID)

	_, err = aws.PIDFromTask(task, "baz")
	assert.Error(t, err)
}

func TestPIDIsMatchingTask(t *testing.T) {
	pid := makeTestPID()

	assert.True(t, pid.IsMatchingTask("123e4567-e89b-12d3-a456-444444444444"))
	assert.False(t, pid.IsMatchingTask("123e4567-e89b-12d3-a456-666666666666"))
}

func TestPIDIsMatchingContainer(t *testing.T) {
	pid := makeTestPID()

	assert.True(t, pid.IsMatchingContainer("123e4567-e89b-12d3-a456-444444444444", "123e4567-e89b-12d3-a456-555555555555"))
	assert.False(t, pid.IsMatchingContainer("123e4567-e89b-12d3-a456-444444444444", "123e4567-e89b-12d3-a456-666666666666"))
}

func TestPIDFindMatchingContainerInTask(t *testing.T) {
	task := makeFakeECSTask()

	pid := makeTestPID()

	container1 := pid.FindMatchingContainerInTask(task)

	assert.NotNil(t, container1)
	assert.Equal(t, "foo", *container1.Name)

	pid.ContainerID = "666666666666"

	container2 := pid.FindMatchingContainerInTask(task)
	assert.Nil(t, container2)
}
