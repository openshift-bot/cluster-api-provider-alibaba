/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machine

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"

	"k8s.io/klog"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	mapierrors "github.com/openshift/machine-api-operator/pkg/controller/machine"

	machinev1 "github.com/openshift/api/machine/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	alibabacloudClient "github.com/openshift/cluster-api-provider-alibaba/pkg/client"
	"github.com/openshift/machine-api-operator/pkg/metrics"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// EcsImageStatusAvailable Image status
	EcsImageStatusAvailable = "Available"

	// MaxInstanceOfSecurityGroupTypeNormal A basic security group can contain a maximum of 2,000 instances.
	MaxInstanceOfSecurityGroupTypeNormal = 2000

	// MaxInstanceOfSecurityGroupTypeEnterprise An advanced security group can contain a maximum of 65,536 instances.
	MaxInstanceOfSecurityGroupTypeEnterprise = 65536

	// SecurityGroupTypeNormal SecurityGroup type normal
	SecurityGroupTypeNormal = "normal"
	// SecurityGroupTypeEnterprise SecurityGroup type enterprise
	SecurityGroupTypeEnterprise = "enterprise"

	// InstanceDefaultTimeout default timeout
	InstanceDefaultTimeout = 900
	// DefaultWaitForInterval default interval
	DefaultWaitForInterval = 5

	// ECSInstanceStatusPending ecs instance status Pedding
	ECSInstanceStatusPending = "Pending"
	// ECSInstanceStatusStarting ecs instance status Starting
	ECSInstanceStatusStarting = "Starting"
	// ECSInstanceStatusRunning ecs instance status Running
	ECSInstanceStatusRunning = "Running"
	// ECSInstanceStatusStopping ecs instance status Stopping
	ECSInstanceStatusStopping = "Stopping"
	// ECSInstanceStatusStopped ecs instance status Stopped
	ECSInstanceStatusStopped = "Stopped"

	// ECSTagResourceTypeInstance  tag resource type
	ECSTagResourceTypeInstance = "instance"
)

// runInstances create ecs
func runInstances(machine *machinev1beta1.Machine, machineProviderConfig *machinev1.AlibabaCloudMachineProviderConfig, userData string, client alibabacloudClient.Client) (*ecs.Instance, error) {
	machineKey := runtimeclient.ObjectKey{
		Name:      machine.Name,
		Namespace: machine.Namespace,
	}

	// ImageID
	imageID, err := getImageID(machineKey, machineProviderConfig, client)
	if err != nil {
		return nil, mapierrors.InvalidMachineConfiguration("error getting ImageID: %v", err)
	}

	// SecurgityGroupIds
	securityGroupIDs, err := getSecurityGroupIDs(machineKey, machineProviderConfig, client)
	if err != nil {
		return nil, mapierrors.InvalidMachineConfiguration("error getting security groups ID: %v", err)
	}

	// VSwitchID
	vSwitchID, err := getVSwitchID(machineKey, machineProviderConfig, client)
	if err != nil {
		return nil, mapierrors.InvalidMachineConfiguration("error getting vswitch ID: %v", err)
	}

	clusterID, ok := getClusterID(machine)
	if !ok {
		klog.Errorf("Unable to get cluster ID for machine: %q", machine.Name)
		return nil, mapierrors.InvalidMachineConfiguration("Unable to get cluster ID for machine: %q", machine.Name)
	}

	// RunInstanceRequest init request params
	runInstancesRequest := ecs.CreateRunInstancesRequest()
	// Scheme, set to https
	runInstancesRequest.Scheme = "https"

	// RegionID
	runInstancesRequest.RegionId = machineProviderConfig.RegionID

	// ResourceGroupID
	if machineProviderConfig.ResourceGroupID != "" {
		runInstancesRequest.ResourceGroupId = machineProviderConfig.ResourceGroupID
	}

	// SecurityGroupIDs
	runInstancesRequest.SecurityGroupIds = securityGroupIDs

	// Add tags to the created machine
	tagList := buildTagList(machine.Name, clusterID, machineProviderConfig.Tags)

	// Tags
	runInstancesRequest.Tag = covertToRunInstancesTag(tagList)

	// ImageID
	runInstancesRequest.ImageId = imageID

	// InstanceType
	runInstancesRequest.InstanceType = machineProviderConfig.InstanceType

	// InstanceName
	runInstancesRequest.InstanceName = machine.GetName()

	// HostName
	runInstancesRequest.HostName = machine.GetName()

	// Amount
	runInstancesRequest.Amount = requests.NewInteger(1)

	// MinAmount
	runInstancesRequest.MinAmount = requests.NewInteger(1)

	// RAMRoleName
	if machineProviderConfig.RAMRoleName != "" {
		runInstancesRequest.RamRoleName = machineProviderConfig.RAMRoleName
	}

	// InternetMaxBandwidthOut
	if machineProviderConfig.Bandwidth.InternetMaxBandwidthOut > 0 {
		runInstancesRequest.InternetMaxBandwidthOut = requests.NewInteger64(machineProviderConfig.Bandwidth.InternetMaxBandwidthOut)
	}

	// InternetMaxBandwidthIn
	if machineProviderConfig.Bandwidth.InternetMaxBandwidthIn != 0 {
		runInstancesRequest.InternetMaxBandwidthIn = requests.NewInteger64(machineProviderConfig.Bandwidth.InternetMaxBandwidthIn)
	}

	// VswitchId
	runInstancesRequest.VSwitchId = vSwitchID

	// SystemDisk
	runInstancesRequest.SystemDiskCategory = machineProviderConfig.SystemDisk.Category
	runInstancesRequest.SystemDiskSize = strconv.FormatInt(machineProviderConfig.SystemDisk.Size, 10)
	if machineProviderConfig.SystemDisk.Name != "" {
		runInstancesRequest.SystemDiskDiskName = machineProviderConfig.SystemDisk.Name
	}

	if machineProviderConfig.SystemDisk.PerformanceLevel != "" {
		runInstancesRequest.SystemDiskPerformanceLevel = machineProviderConfig.SystemDisk.PerformanceLevel
	}

	// DataDisk
	if len(machineProviderConfig.DataDisks) > 0 {
		dataDisks := make([]ecs.RunInstancesDataDisk, 0)
		for _, dataDisk := range machineProviderConfig.DataDisks {
			runInstancesDataDisk := ecs.RunInstancesDataDisk{
				Size:      strconv.FormatInt(dataDisk.Size, 10),
				Category:  string(dataDisk.Category),
				Encrypted: strconv.FormatBool(dataDisk.DiskEncryption == machinev1.AlibabaDiskEncryptionEnabled),
			}
			// DiskName
			if dataDisk.Name != "" {
				runInstancesDataDisk.DiskName = dataDisk.Name
			}

			// SnapshotID
			if dataDisk.SnapshotID != "" {
				runInstancesDataDisk.SnapshotId = dataDisk.SnapshotID
			}

			// PerformanceLevel
			if dataDisk.PerformanceLevel != "" {
				runInstancesDataDisk.PerformanceLevel = string(dataDisk.PerformanceLevel)
			}

			// KMSKeyID
			if dataDisk.KMSKeyID != "" {
				runInstancesDataDisk.KMSKeyId = dataDisk.KMSKeyID
			}

			// DeleteWithInstance
			if dataDisk.DiskPreservation == machinev1.DeleteWithInstance {
				runInstancesDataDisk.DeleteWithInstance = strconv.FormatBool(true)
			}

			dataDisks = append(dataDisks, runInstancesDataDisk)
		}
		runInstancesRequest.DataDisk = &dataDisks
	}

	if userData != "" {
		runInstancesRequest.UserData = userData
	}

	// Setting Tenancy
	instanceTenancy := machineProviderConfig.Tenancy

	switch instanceTenancy {
	case "":
		// Set DefaultTenancy  when not set
		runInstancesRequest.Tenancy = string(machinev1.DefaultTenancy)
	case machinev1.DefaultTenancy, machinev1.HostTenancy:
		runInstancesRequest.Tenancy = string(instanceTenancy)
	default:
		return nil, mapierrors.CreateMachine("invalid instance tenancy: %s. Allowed options are: %s,%s",
			instanceTenancy,
			machinev1.DefaultTenancy,
			machinev1.HostTenancy)
	}
	runResponse, err := client.RunInstances(runInstancesRequest)
	if err != nil {
		metrics.RegisterFailedInstanceCreate(&metrics.MachineLabels{
			Name:      machine.Name,
			Namespace: machine.Namespace,
			Reason:    err.Error(),
		})

		klog.Errorf("Error creating ECS instance: %v", err)
		return nil, mapierrors.CreateMachine("error creating ECS instance: %v", err)
	}

	if runResponse == nil || len(runResponse.InstanceIdSets.InstanceIdSet) != 1 {
		klog.Errorf("Unexpected reservation creating instances: %v", runResponse)
		return nil, mapierrors.CreateMachine("unexpected reservation creating instance")
	}

	// Sleep
	time.Sleep(5 * time.Second)

	// Query the status of the instance until Running
	instance, err := waitForInstancesStatus(client, machineProviderConfig.RegionID, []string{runResponse.InstanceIdSets.InstanceIdSet[0]}, ECSInstanceStatusRunning, InstanceDefaultTimeout)
	if err != nil {
		metrics.RegisterFailedInstanceCreate(&metrics.MachineLabels{
			Name:      machine.Name,
			Namespace: machine.Namespace,
			Reason:    err.Error(),
		})

		klog.Errorf("Error waiting ECS instance to Running: %v", err)
		return nil, mapierrors.CreateMachine("error waiting ECS instance to Running: %v", err)
	}

	if instance == nil || len(instance) < 1 {
		return nil, mapierrors.CreateMachine(" ECS instance %s not found", runResponse.InstanceIdSets.InstanceIdSet[0])
	}

	return instance[0], nil
}

// waitForInstancesStatus waits for instances to given status when instance.NotFound wait until timeout
func waitForInstancesStatus(client alibabacloudClient.Client, regionID string, instanceIds []string, instanceStatus string, timeout int) ([]*ecs.Instance, error) {
	if timeout <= 0 {
		timeout = InstanceDefaultTimeout
	}

	result, err := WaitForResult(fmt.Sprintf("Wait for the instances %v state to change to %s ", instanceIds, instanceStatus), func() (stop bool, result interface{}, err error) {
		describeInstancesRequest := ecs.CreateDescribeInstancesRequest()
		describeInstancesRequest.RegionId = regionID
		ids, _ := json.Marshal(instanceIds)
		describeInstancesRequest.InstanceIds = string(ids)
		describeInstancesRequest.Scheme = "https"
		describeInstancesResponse, err := client.DescribeInstances(describeInstancesRequest)
		klog.V(3).Infof("instance resonpse %v", describeInstancesResponse)
		if err != nil {
			return false, nil, err
		}

		if len(describeInstancesResponse.Instances.Instance) <= 0 {
			return true, nil, fmt.Errorf("the instances %v not found. ", instanceIds)
		}

		idsLen := len(instanceIds)
		instances := make([]*ecs.Instance, 0)

		for _, instance := range describeInstancesResponse.Instances.Instance {
			if instance.Status == instanceStatus {
				instances = append(instances, &instance)
			}
		}

		if len(instances) == idsLen {
			return true, instances, nil
		}

		return false, nil, fmt.Errorf("the instances  %v state are not  the expected state  %s ", instanceIds, instanceStatus)

	}, false, DefaultWaitForInterval, timeout)

	if err != nil {
		klog.Errorf("Wait for the instances %v state change to %v occur error %v", instanceIds, instanceStatus, err)
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return result.([]*ecs.Instance), nil
}

func getImageID(machine runtimeclient.ObjectKey, machineProviderConfig *machinev1.AlibabaCloudMachineProviderConfig, client alibabacloudClient.Client) (string, error) {
	klog.Infof("%s validate image in region %s", machineProviderConfig.ImageID, machineProviderConfig.RegionID)
	request := ecs.CreateDescribeImagesRequest()
	request.ImageId = machineProviderConfig.ImageID
	request.RegionId = machineProviderConfig.RegionID
	request.ShowExpired = requests.NewBoolean(true)
	request.Scheme = "https"

	response, err := client.DescribeImages(request)
	if err != nil {
		metrics.RegisterFailedInstanceCreate(&metrics.MachineLabels{
			Name:      machine.Name,
			Namespace: machine.Namespace,
			Reason:    err.Error(),
		})
		klog.Errorf("error describing Image: %v", err)
		return "", fmt.Errorf("error describing Images: %v", err)
	}

	if len(response.Images.Image) < 1 {
		klog.Errorf("no image for given filters not found")
		return "", fmt.Errorf("no image for given filters not found")
	}

	image := response.Images.Image[0]
	if image.Status != EcsImageStatusAvailable {
		klog.Errorf("%s invalid image status: %s", machineProviderConfig.ImageID, image.Status)
		return "", fmt.Errorf("%s invalid image status: %s", machineProviderConfig.ImageID, image.Status)
	}

	return image.ImageId, nil
}

func getSecurityGroupIDs(machine runtimeclient.ObjectKey, machineProviderConfig *machinev1.AlibabaCloudMachineProviderConfig, client alibabacloudClient.Client) (*[]string, error) {
	klog.Infof("query security groups in region %s", machineProviderConfig.RegionID)
	var securityGroupIDs []string

	// If SecurityGroupID is assigned, use it directly
	if len(machineProviderConfig.SecurityGroups) == 0 {
		return nil, errors.New("no security configuration provided")
	}

	for _, sg := range machineProviderConfig.SecurityGroups {
		if sg.ID != "" {
			securityGroupIDs = append(securityGroupIDs, sg.ID)
		} else {
			if sg.Tags != nil {
				ids, err := getSecurityGroupIDByTags(machine, machineProviderConfig, sg.Tags, client)
				if err != nil {
					return nil, err
				}
				securityGroupIDs = append(securityGroupIDs, ids...)
			}
		}
	}
	if len(securityGroupIDs) == 0 {
		return nil, errors.New("no securitygroup IDs found from configuration")
	}
	return &securityGroupIDs, nil
}

func getSecurityGroupIDByTags(machine runtimeclient.ObjectKey, machineProviderConfig *machinev1.AlibabaCloudMachineProviderConfig, tags []machinev1.Tag, client alibabacloudClient.Client) ([]string, error) {
	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.VpcId = machineProviderConfig.VpcID
	request.ResourceGroupId = machineProviderConfig.ResourceGroupID
	request.RegionId = machineProviderConfig.RegionID
	request.Tag = buildDescribeSecurityGroupsTag(tags)
	request.Scheme = "https"

	response, err := client.DescribeSecurityGroups(request)
	if err != nil {
		metrics.RegisterFailedInstanceCreate(&metrics.MachineLabels{
			Name:      machine.Name,
			Namespace: machine.Namespace,
			Reason:    err.Error(),
		})
		klog.Errorf("error describing securitygroup: %v", err)
		return nil, fmt.Errorf("error describing securitygroup: %v", err)
	}
	if len(response.SecurityGroups.SecurityGroup) < 1 {
		klog.Errorf("no securitygroup for given tags not found")
		return nil, fmt.Errorf("no securitygroup for given tags not found")
	}
	securityGroupIDs := []string{}
	for _, sg := range response.SecurityGroups.SecurityGroup {
		securityGroupIDs = append(securityGroupIDs, sg.SecurityGroupId)
	}
	return securityGroupIDs, nil
}

func getMaxInstancesBySecurityGroupType(securityGroupType string) int {
	switch securityGroupType {
	case SecurityGroupTypeNormal:
		return MaxInstanceOfSecurityGroupTypeNormal
	case SecurityGroupTypeEnterprise:
		return MaxInstanceOfSecurityGroupTypeEnterprise
	default:
		return MaxInstanceOfSecurityGroupTypeNormal
	}
}

func buildDescribeSecurityGroupsTag(tags []machinev1.Tag) *[]ecs.DescribeSecurityGroupsTag {
	describeSecurityGroupsTag := make([]ecs.DescribeSecurityGroupsTag, len(tags))

	for index, tag := range tags {
		describeSecurityGroupsTag[index] = ecs.DescribeSecurityGroupsTag{
			Key:   tag.Key,
			Value: tag.Value,
		}
	}

	return &describeSecurityGroupsTag
}

func getVSwitchID(machine runtimeclient.ObjectKey, machineProviderConfig *machinev1.AlibabaCloudMachineProviderConfig, client alibabacloudClient.Client) (string, error) {
	klog.Infof("validate vswitch in region %s", machineProviderConfig.RegionID)
	if machineProviderConfig.VSwitch.ID == "" && len(machineProviderConfig.VSwitch.Tags) == 0 {
		return "", errors.New("no vswitch configuration provided")
	}

	if machineProviderConfig.VSwitch.ID != "" {
		return machineProviderConfig.VSwitch.ID, nil
	}

	if machineProviderConfig.VSwitch.Tags != nil {
		return getVSwitchIDFromTags(machine, machineProviderConfig, client)
	}

	return "", fmt.Errorf("no vSwitch found from configuration")
}

func getVSwitchIDFromTags(machine runtimeclient.ObjectKey, mpc *machinev1.AlibabaCloudMachineProviderConfig, client alibabacloudClient.Client) (string, error) {
	// Build a request to fetch the vSwitchID from the tags provided
	describeVSwitchesRequest := vpc.CreateDescribeVSwitchesRequest()
	describeVSwitchesRequest.Scheme = "https"
	describeVSwitchesRequest.RegionId = mpc.RegionID
	describeVSwitchesRequest.VpcId = mpc.VpcID
	describeVSwitchesRequest.Tag = buildDescribeVSwitchesTag(mpc.VSwitch.Tags)
	describeVSwitchesResponse, err := client.DescribeVSwitches(describeVSwitchesRequest)
	if err != nil {
		metrics.RegisterFailedInstanceCreate(&metrics.MachineLabels{
			Name:      machine.Name,
			Namespace: machine.Namespace,
			Reason:    err.Error(),
		})
		klog.Errorf("error describing vswitches: %v", err)
		return "", fmt.Errorf("error describing vswitches: %v", err)
	}
	if len(describeVSwitchesResponse.VSwitches.VSwitch) < 1 {
		klog.Errorf("no vswitches found for given tags, vpcid, and regionid")
		return "", fmt.Errorf("no vswitches found for given tags, vpcid, and regionid")
	}
	return describeVSwitchesResponse.VSwitches.VSwitch[0].VSwitchId, nil
}

func buildDescribeVSwitchesTag(tags []machinev1.Tag) *[]vpc.DescribeVSwitchesTag {
	describeVSwitchesTag := make([]vpc.DescribeVSwitchesTag, len(tags))

	for index, tag := range tags {
		describeVSwitchesTag[index] = vpc.DescribeVSwitchesTag{
			Key:   tag.Key,
			Value: tag.Value,
		}
	}

	return &describeVSwitchesTag
}

// buildTagList compile a list of ecs tags from machine provider spec and infrastructure object platform spec
func buildTagList(machineName string, clusterID string, machineTags []machinev1.Tag) []*machinev1.Tag {
	rawTagList := make([]*machinev1.Tag, 0)

	for _, tag := range machineTags {
		// Alibabacoud tags are case sensitive, so we don't need to worry about other casing of "Name"
		if !strings.HasPrefix(tag.Key, clusterFilterKeyPrefix) && tag.Key != clusterFilterName {
			rawTagList = append(rawTagList, &machinev1.Tag{Key: tag.Key, Value: tag.Value})
		}
	}
	rawTagList = append(rawTagList, []*machinev1.Tag{
		{Key: clusterFilterKeyPrefix + clusterID, Value: clusterFilterValue},
		{Key: clusterFilterName, Value: machineName},
		{Key: clusterOwnedKey, Value: clusterOwnedValue},
		{Key: machineTagKeyFrom, Value: machineTagValueFrom},
		{Key: machineIsvIntegrationTagKey, Value: machineTagValueFrom},
	}...)

	return removeDuplicatedTags(rawTagList)
}

// Scan machine tags, and return a deduped tags list. The first found value gets precedence.
func removeDuplicatedTags(tags []*machinev1.Tag) []*machinev1.Tag {
	m := make(map[string]bool)
	result := make([]*machinev1.Tag, 0)

	// look for duplicates
	for _, entry := range tags {
		if _, value := m[entry.Key]; !value {
			m[entry.Key] = true
			result = append(result, entry)
		}
	}
	return result
}

func covertToRunInstancesTag(tags []*machinev1.Tag) *[]ecs.RunInstancesTag {
	runInstancesTags := make([]ecs.RunInstancesTag, 0)

	for _, tag := range tags {
		runInstancesTags = append(runInstancesTags, ecs.RunInstancesTag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}

	return &runInstancesTags
}

func getExistingInstanceByID(instanceID string, regionID string, client alibabacloudClient.Client) (*ecs.Instance, error) {
	return getInstanceByID(instanceID, regionID, client, supportedInstanceStates())
}

// getInstanceByID returns the instance with the given ID if it exists.
func getInstanceByID(instanceID string, regionID string, client alibabacloudClient.Client, instanceStates []string) (*ecs.Instance, error) {
	if instanceID == "" {
		return nil, fmt.Errorf("instance-id not specified")
	}

	instances, err := describeInstances([]string{instanceID}, regionID, client)
	if err != nil {
		return nil, err
	}
	if len(instances) != 1 {
		return nil, fmt.Errorf("found %d instances for instance-id %s", len(instances), instanceID)
	}

	instance := instances[0]

	return &instance, instanceHasSupportedState(&instance, instanceStates)
}

func describeInstances(instanceIds []string, regionID string, client alibabacloudClient.Client) ([]ecs.Instance, error) {
	if len(instanceIds) < 1 {
		return nil, fmt.Errorf("instance-ids not specified")
	}

	describeInstancesRequest := ecs.CreateDescribeInstancesRequest()
	describeInstancesRequest.RegionId = regionID
	describeInstancesRequest.Scheme = "https"
	instancesIds, _ := json.Marshal(instanceIds)
	describeInstancesRequest.InstanceIds = string(instancesIds)

	result, err := client.DescribeInstances(describeInstancesRequest)
	if err != nil {
		return nil, err
	}

	return result.Instances.Instance, nil
}

func instanceHasSupportedState(instance *ecs.Instance, instanceStates []string) error {
	if instance.InstanceId == "" {
		return fmt.Errorf("instance has nil ID")
	}

	if instance.Status == "" {
		return fmt.Errorf("instance %s has nil state", instance.InstanceId)
	}

	if len(instanceStates) == 0 {
		return nil
	}

	actualState := instance.Status
	for _, supportedState := range instanceStates {
		if supportedState == actualState {
			return nil
		}
	}

	supportedStates := make([]string, 0, len(instanceStates))
	for _, allowedState := range instanceStates {
		supportedStates = append(supportedStates, allowedState)
	}
	return fmt.Errorf("instance %s state %q is not in %s", instance.InstanceId, actualState, strings.Join(supportedStates, ", "))
}

// getExistingInstances returns all instances not terminated
func getExistingInstances(machine *machinev1beta1.Machine, regionID string, client alibabacloudClient.Client) ([]*ecs.Instance, error) {
	return getInstances(machine, regionID, client, supportedInstanceStates())
}

// getInstances returns all instances that have a tag matching our machine name,
// and cluster ID.
func getInstances(machine *machinev1beta1.Machine, regionID string, client alibabacloudClient.Client, instanceStates []string) ([]*ecs.Instance, error) {
	clusterID, ok := getClusterID(machine)
	if !ok {
		return nil, fmt.Errorf("unable to get cluster ID for machine: %q", machine.Name)
	}

	request := ecs.CreateDescribeInstancesRequest()
	request.RegionId = regionID
	describeInstancesTags := []ecs.DescribeInstancesTag{
		{Key: clusterFilterKeyPrefix + clusterID, Value: clusterFilterValue},
		{Key: clusterFilterName, Value: machine.Name},
	}

	request.Tag = &describeInstancesTags

	result, err := client.DescribeInstances(request)
	if err != nil {
		return nil, err
	}

	instances := make([]*ecs.Instance, 0)

	for _, instance := range result.Instances.Instance {
		err := instanceHasSupportedState(&instance, instanceStates)
		if err != nil {
			klog.Errorf("Excluding instance matching %s: %v", machine.Name, err)
		} else {
			instances = append(instances, &instance)
		}
	}

	return instances, nil
}

// stopInstances stop all provided instances with a single ECS request.
func stopInstances(client alibabacloudClient.Client, regionID string, instances []*ecs.Instance) ([]ecs.InstanceResponse, error) {
	instanceIDs := make([]string, 0)
	// Stop all older instances:
	for _, instance := range instances {
		klog.Infof("Cleaning up extraneous instance for machine: %v, state: %v, launchTime: %v", instance.InstanceId, instance.Status, instance.StartTime)
		instanceIDs = append(instanceIDs, instance.InstanceId)
	}

	// Describe instances ,only running instance can be stopped
	existingInstances, err := describeInstances(instanceIDs, regionID, client)
	if err != nil {
		klog.Errorf("failed to describe instances %v", err)
		return nil, err
	}

	if len(existingInstances) < 1 {
		return nil, fmt.Errorf("instances %v not exist", instanceIDs)
	}

	// needStoppedInstance
	needStoppedInstanceIDs := make([]string, 0)
	for _, instance := range existingInstances {
		if instance.Status == ECSInstanceStatusRunning {
			needStoppedInstanceIDs = append(needStoppedInstanceIDs, instance.InstanceId)
		}
	}

	for _, instanceID := range needStoppedInstanceIDs {
		klog.Infof("Stopping %v instance", instanceID)
	}

	stopInstancesRequest := ecs.CreateStopInstancesRequest()
	stopInstancesRequest.RegionId = regionID
	stopInstancesRequest.Scheme = "https"
	stopInstancesRequest.InstanceId = &needStoppedInstanceIDs

	stopInstancesResponse, err := client.StopInstances(stopInstancesRequest)
	if err != nil {
		klog.Errorf("Error stopping instances: %v", err)
		return nil, fmt.Errorf("error stopping instances: %v", err)
	}

	if stopInstancesResponse == nil {
		return nil, nil
	}

	return stopInstancesResponse.InstanceResponses.InstanceResponse, nil
}

type instanceList []*ecs.Instance

func (il instanceList) Len() int {
	return len(il)
}

func (il instanceList) Swap(i, j int) {
	il[i], il[j] = il[j], il[i]
}

const formatISO8601 = "2006-01-02T15:04:05Z"

func (il instanceList) Less(i, j int) bool {
	if il[i].StartTime == "" && il[j].StartTime == "" {
		return false
	}
	if il[i].StartTime != "" && il[j].StartTime == "" {
		return false
	}
	if il[i].StartTime == "" && il[j].StartTime != "" {
		return true
	}

	iStartTime, err := time.ParseInLocation(formatISO8601, il[i].StartTime, time.Local)
	if err != nil {
		return false
	}

	jStartTime, err := time.ParseInLocation(formatISO8601, il[j].StartTime, time.Local)
	if err != nil {
		return false
	}

	return iStartTime.After(jStartTime)
}

// sortInstances will sort a list of instance based on an instace launch time
// from the newest to the oldest.
// This function should only be called with running instances, not those which are stopped or
// terminated.
func sortInstances(instances []*ecs.Instance) {
	sort.Sort(instanceList(instances))
}

// getRunningFromInstances returns all running instances from a list of instances.
func getRunningFromInstances(instances []*ecs.Instance) []*ecs.Instance {
	var runningInstances []*ecs.Instance
	for _, instance := range instances {
		if instance.Status == ECSInstanceStatusRunning {
			runningInstances = append(runningInstances, instance)
		}
	}
	return runningInstances
}

// correctExistingTags validates Name and clusterID tags are correct on the instance
// and sets them if they are not.
func correctExistingTags(machine *machinev1beta1.Machine, regionID string, instance *ecs.Instance, client alibabacloudClient.Client) error {
	// https://www.alibabacloud.com/help/en/doc-detail/110424.htm
	if instance == nil || instance.InstanceId == "" {
		return fmt.Errorf("unexpected nil found in instance: %v", instance)
	}
	clusterID, ok := getClusterID(machine)
	if !ok {
		return fmt.Errorf("unable to get cluster ID for machine: %q", machine.Name)
	}
	nameTagOk := false
	clusterTagOk := false
	ownedTagOk := false
	for _, tag := range instance.Tags.Tag {
		if tag.TagKey != "" && tag.TagValue != "" {
			if tag.TagKey == clusterFilterName && tag.TagValue == machine.Name {
				nameTagOk = true
			}
			if tag.TagKey == clusterFilterKeyPrefix+clusterID && tag.TagValue == clusterFilterValue {
				clusterTagOk = true
			}
			if tag.TagKey == clusterOwnedKey && tag.TagValue == clusterOwnedValue {
				ownedTagOk = true
			}
		}
	}

	// Update our tags if they're not set or correct
	if !nameTagOk || !clusterTagOk || !ownedTagOk {
		// Create tags only adds/replaces what is present, does not affect other tags.
		request := ecs.CreateTagResourcesRequest()
		request.Scheme = "https"
		request.RegionId = regionID
		request.Tag = tagResourceTags(clusterID, machine.Name)
		request.ResourceId = &[]string{instance.InstanceId}
		request.ResourceType = ECSTagResourceTypeInstance

		klog.Infof("Invalid or missing instance tags for machine: %v; instanceID: %v, updating", machine.Name, instance.InstanceId)
		_, err := client.TagResources(request)
		return err
	}

	return nil
}
