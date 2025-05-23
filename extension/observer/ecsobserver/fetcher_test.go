// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ecsobserver

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/ecsobserver/internal/ecsmock"
)

func TestFetcher_FetchAndDecorate(t *testing.T) {
	c := ecsmock.NewCluster()
	f := newTestTaskFetcher(t, c, c)
	// Create 1 task def, 2 services and 10 tasks, 8 running on ec2, first 3 runs on fargate
	nTasks := 11
	nInstances := 2
	nFargateInstances := 3
	c.SetTaskDefinitions(ecsmock.GenTaskDefinitions("d", 1, 1, nil))
	c.SetTasks(ecsmock.GenTasks("t", nTasks, func(i int, task *ecstypes.Task) {
		ins := i % nInstances
		if i < nFargateInstances {
			task.LaunchType = ecstypes.LaunchTypeFargate
			task.StartedBy = aws.String("deploy0")
		} else {
			task.LaunchType = ecstypes.LaunchTypeEc2
			task.ContainerInstanceArn = aws.String(fmt.Sprintf("ci%d", ins))
			task.StartedBy = aws.String("deploy1")
		}
		task.TaskDefinitionArn = aws.String("d0:1")
	}))
	c.SetEc2Instances([]ec2types.Instance{
		{InstanceId: aws.String("i-0")},
		{InstanceId: aws.String("i-1")},
	})
	// Setting container instance and ec2 is same as previous sub test
	c.SetContainerInstances(ecsmock.GenContainerInstances("ci", nInstances, func(i int, ci *ecstypes.ContainerInstance) {
		ci.Ec2InstanceId = aws.String(fmt.Sprintf("i-%d", i))
	}))
	// Service
	c.SetServices(ecsmock.GenServices("s", 2, func(i int, s *ecstypes.Service) {
		if i == 0 {
			s.LaunchType = ecstypes.LaunchTypeFargate
			s.Deployments = []ecstypes.Deployment{
				{
					Status: aws.String("ACTIVE"),
					Id:     aws.String("deploy0"),
				},
			}
		} else {
			s.LaunchType = ecstypes.LaunchTypeEc2
			s.Deployments = []ecstypes.Deployment{
				{
					Status: aws.String("ACTIVE"),
					Id:     aws.String("deploy1"),
				},
			}
		}
	}))

	tasks, err := f.fetchAndDecorate(context.Background())
	require.NoError(t, err)
	assert.Len(t, tasks, nTasks)
	assert.Equal(t, "s0", *tasks[0].Service.ServiceArn)
}

func TestFetcher_GetDiscoverableTasks(t *testing.T) {
	t.Run("without non discoverable tasks", func(t *testing.T) {
		c := ecsmock.NewCluster()
		f := newTestTaskFetcher(t, c, c)
		const nTasks = 203
		c.SetTasks(ecsmock.GenTasks("p", nTasks, nil))
		ctx := context.Background()
		tasks, err := f.getDiscoverableTasks(ctx)
		require.NoError(t, err)
		assert.Len(t, tasks, nTasks)
	})

	t.Run("with non discoverable tasks", func(t *testing.T) {
		c := ecsmock.NewCluster()
		f := newTestTaskFetcher(t, c, c)
		nTasks := 3

		c.SetTaskDefinitions(ecsmock.GenTaskDefinitions("d", 1, 1, nil))
		c.SetTasks(ecsmock.GenTasks("t", nTasks, func(i int, task *ecstypes.Task) {
			task.TaskDefinitionArn = aws.String("d0:1")
			switch i {
			case 0:
				task.LaunchType = ecstypes.LaunchTypeEc2
				task.ContainerInstanceArn = nil
			case 1:
				task.LaunchType = ecstypes.LaunchTypeFargate
			case 2:
				task.LaunchType = ecstypes.LaunchTypeEc2
				task.ContainerInstanceArn = aws.String("ci0")
			}
		}))

		ctx := context.Background()
		tasks, err := f.getDiscoverableTasks(ctx)
		require.NoError(t, err)

		// Expect 2 tasks, with LaunchType Fargate and EC2 with non-nil ContainerInstanceArn
		assert.Len(t, tasks, 2)
		assert.Equal(t, ecstypes.LaunchTypeFargate, tasks[0].LaunchType)
		assert.Equal(t, ecstypes.LaunchTypeEc2, tasks[1].LaunchType)
	})
}

func TestFetcher_AttachTaskDefinitions(t *testing.T) {
	c := ecsmock.NewCluster()
	f := newTestTaskFetcher(t, c, c)

	const nTasks = 5
	ctx := context.Background()
	// one task per def
	c.SetTasks(ecsmock.GenTasks("p", nTasks, func(i int, task *ecstypes.Task) {
		task.TaskDefinitionArn = aws.String(fmt.Sprintf("pdef%d:1", i))
	}))
	c.SetTaskDefinitions(ecsmock.GenTaskDefinitions("pdef", nTasks, 1, nil))

	// no cache
	tasks, err := f.getDiscoverableTasks(ctx)
	require.NoError(t, err)
	attached, err := f.attachTaskDefinition(ctx, tasks)
	stats := c.Stats()
	require.NoError(t, err)
	assert.Len(t, attached, len(tasks))
	assert.Equal(t, nTasks, stats.DescribeTaskDefinition.Called)

	// all cached
	tasks, err = f.getDiscoverableTasks(ctx)
	require.NoError(t, err)
	// do it again to trigger cache logic
	attached, err = f.attachTaskDefinition(ctx, tasks)
	stats = c.Stats()
	require.NoError(t, err)
	assert.Len(t, attached, len(tasks))
	assert.Equal(t, nTasks, stats.DescribeTaskDefinition.Called) // no api call due to cache

	// add a new task
	c.SetTasks(ecsmock.GenTasks("p", nTasks+1, func(i int, task *ecstypes.Task) {
		task.TaskDefinitionArn = aws.String(fmt.Sprintf("pdef%d:1", i))
	}))
	c.SetTaskDefinitions(ecsmock.GenTaskDefinitions("pdef", nTasks+1, 1, nil))
	tasks, err = f.getDiscoverableTasks(ctx)
	require.NoError(t, err)
	_, err = f.attachTaskDefinition(ctx, tasks)
	stats = c.Stats()
	require.NoError(t, err)
	assert.Equal(t, nTasks+1, stats.DescribeTaskDefinition.Called) // +1 for new task
}

func TestFetcher_AttachContainerInstance(t *testing.T) {
	t.Run("ec2 only", func(t *testing.T) {
		c := ecsmock.NewCluster()
		f := newTestTaskFetcher(t, c, c)
		// Create 1 task def and 11 tasks running on 2 ec2 instances
		nTasks := 11
		nInstances := 2
		c.SetTaskDefinitions(ecsmock.GenTaskDefinitions("d", 1, 1, nil))
		c.SetTasks(ecsmock.GenTasks("t", nTasks, func(i int, task *ecstypes.Task) {
			ins := i % nInstances
			task.LaunchType = ecstypes.LaunchTypeEc2
			task.TaskDefinitionArn = aws.String("d0:1")
			task.ContainerInstanceArn = aws.String(fmt.Sprintf("ci%d", ins))
		}))
		c.SetContainerInstances(ecsmock.GenContainerInstances("ci", nInstances, func(i int, ci *ecstypes.ContainerInstance) {
			ci.Ec2InstanceId = aws.String(fmt.Sprintf("i-%d", i))
		}))
		c.SetEc2Instances(ecsmock.GenEc2Instances("i-", nInstances, nil))

		ctx := context.Background()
		rawTasks, err := f.getDiscoverableTasks(ctx)
		require.NoError(t, err)
		assert.Len(t, rawTasks, nTasks)

		tasks, err := f.attachTaskDefinition(ctx, rawTasks)
		require.NoError(t, err)
		assert.Equal(t, "d0:1", *tasks[0].Definition.TaskDefinitionArn)

		err = f.attachContainerInstance(ctx, tasks)
		require.NoError(t, err)
		assert.Equal(t, "i-0", *tasks[0].EC2.InstanceId)
	})

	t.Run("mixed cluster", func(t *testing.T) {
		c := ecsmock.NewCluster()
		f := newTestTaskFetcher(t, c, c)
		// Create 1 task def and 10 tasks, 8 running on ec2, first 3 runs on fargate
		nTasks := 11
		nInstances := 2
		nFargateInstances := 3
		c.SetTaskDefinitions(ecsmock.GenTaskDefinitions("d", 1, 1, nil))
		c.SetTasks(ecsmock.GenTasks("t", nTasks, func(i int, task *ecstypes.Task) {
			ins := i % nInstances
			if i < nFargateInstances {
				task.LaunchType = ecstypes.LaunchTypeFargate
			} else {
				task.LaunchType = ecstypes.LaunchTypeEc2
				task.ContainerInstanceArn = aws.String(fmt.Sprintf("ci%d", ins))
			}
			task.TaskDefinitionArn = aws.String("d0:1")
		}))
		// Setting container instance and ec2 is same as previous sub test
		c.SetContainerInstances(ecsmock.GenContainerInstances("ci", nInstances, func(i int, ci *ecstypes.ContainerInstance) {
			ci.Ec2InstanceId = aws.String(fmt.Sprintf("i-%d", i))
		}))
		c.SetEc2Instances(ecsmock.GenEc2Instances("i-", nInstances, nil))

		ctx := context.Background()
		rawTasks, err := f.getDiscoverableTasks(ctx)
		require.NoError(t, err)
		assert.Len(t, rawTasks, nTasks)

		tasks, err := f.attachTaskDefinition(ctx, rawTasks)
		require.NoError(t, err)
		assert.Equal(t, "d0:1", *tasks[0].Definition.TaskDefinitionArn)

		err = f.attachContainerInstance(ctx, tasks)
		require.NoError(t, err)
		assert.Nil(t, tasks[0].EC2)
		// task instance pattern is 0 1 0 1 ..., nFargateInstances = 3 so the 4th task is running on instance 1
		assert.Equal(t, "i-1", *tasks[nFargateInstances].EC2.InstanceId)
	})
}

func TestFetcher_GetAllServices(t *testing.T) {
	c := ecsmock.NewCluster()
	f := newTestTaskFetcher(t, c, c)
	const nServices = 101
	c.SetServices(ecsmock.GenServices("s", nServices, nil))
	ctx := context.Background()
	services, err := f.getAllServices(ctx)
	require.NoError(t, err)
	assert.Len(t, services, nServices)
}

func TestFetcher_AttachService(t *testing.T) {
	c := ecsmock.NewCluster()
	f := newTestTaskFetcher(t, c, c)
	const nServices = 10
	c.SetServices(ecsmock.GenServices("s", nServices, func(i int, s *ecstypes.Service) {
		s.Deployments = []ecstypes.Deployment{
			{
				Status: aws.String("ACTIVE"),
				Id:     aws.String(fmt.Sprintf("deploy%d", i)),
			},
		}
	}))
	c.SetTaskDefinitions(ecsmock.GenTaskDefinitions("def", nServices, 1, nil))
	const nTasks = 100
	c.SetTasks(ecsmock.GenTasks("t", nTasks, func(i int, task *ecstypes.Task) {
		deployID := i % nServices
		task.TaskDefinitionArn = aws.String(fmt.Sprintf("def%d:1", deployID))
		// Last task is launched manually w/o service
		if i == nTasks-1 {
			return
		}
		task.StartedBy = aws.String(fmt.Sprintf("deploy%d", deployID))
	}))

	ctx := context.Background()
	rawTasks, err := f.getDiscoverableTasks(ctx)
	require.NoError(t, err)
	tasks, err := f.attachTaskDefinition(ctx, rawTasks)
	require.NoError(t, err)
	services, err := f.getAllServices(ctx)
	require.NoError(t, err)
	f.attachService(tasks, services)

	// Just pick one
	assert.Equal(t, "s0", *tasks[0].Service.ServiceArn)
	assert.NotNil(t, tasks[nTasks-2].Service)
	assert.Nil(t, tasks[nTasks-1].Service)
}
