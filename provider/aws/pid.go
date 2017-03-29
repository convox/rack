package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/ecs"
)

func lastSectionOfArn(arn string) string {
	parts := strings.Split(arn, "-")
	return parts[len(parts)-1]
}

// PID - Abstraction to deal with process ids when reference both a task and one of it's containers
type PID struct {
	TaskID      string
	ContainerID string
}

func (pid *PID) String() string {
	return fmt.Sprintf("%s-%s", pid.TaskID, pid.ContainerID)
}

// IsMatchingTask - Tests if the ecs task matches this PID
func (pid *PID) IsMatchingTask(taskArn string) bool {
	return lastSectionOfArn(taskArn) == pid.TaskID
}

// IsMatchingContainer - Tests if the container matches this PID
func (pid *PID) IsMatchingContainer(taskArn string, containerArn string) bool {
	return lastSectionOfArn(taskArn) == pid.TaskID && lastSectionOfArn(containerArn) == pid.ContainerID
}

// FindMatchingContainerInTask - Finds a matching container given the current ecs task
func (pid *PID) FindMatchingContainerInTask(task *ecs.Task) *ecs.Container {
	for _, container := range task.Containers {
		if pid.IsMatchingContainer(*task.TaskArn, *container.ContainerArn) {
			return container
		}
	}
	return nil
}

// ParsePID - Parses a pid into it's constituent parts
func ParsePID(pidStr string) (*PID, error) {
	pidParts := strings.Split(pidStr, "-")
	if len(pidParts) != 2 {
		return nil, fmt.Errorf("Invalid PID string `%s`", pidStr)
	}
	pid := PID{
		TaskID:      pidParts[0],
		ContainerID: pidParts[1],
	}
	return &pid, nil
}

// PIDFromArns - Derives a pid from ecs task and container arns
func PIDFromArns(taskArn string, containerArn string) *PID {
	return &PID{
		TaskID:      lastSectionOfArn(taskArn),
		ContainerID: lastSectionOfArn(containerArn),
	}
}

// PIDFromTask - Derives a pid from am ecs task and the target container name
func PIDFromTask(task *ecs.Task, containerName string) (*PID, error) {
	var containerArn string
	for _, container := range task.Containers {
		if *container.Name == containerName {
			containerArn = *container.ContainerArn
			break
		}
	}
	if containerArn == "" {
		return nil, fmt.Errorf("Cannot find container `%s` in task `%s`", containerName, *task.TaskArn)
	}
	return PIDFromArns(*task.TaskArn, containerArn), nil
}
