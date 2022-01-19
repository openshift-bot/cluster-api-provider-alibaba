package ecs

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// DescribeTaskAttribute invokes the ecs.DescribeTaskAttribute API synchronously
func (client *Client) DescribeTaskAttribute(request *DescribeTaskAttributeRequest) (response *DescribeTaskAttributeResponse, err error) {
	response = CreateDescribeTaskAttributeResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeTaskAttributeWithChan invokes the ecs.DescribeTaskAttribute API asynchronously
func (client *Client) DescribeTaskAttributeWithChan(request *DescribeTaskAttributeRequest) (<-chan *DescribeTaskAttributeResponse, <-chan error) {
	responseChan := make(chan *DescribeTaskAttributeResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeTaskAttribute(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// DescribeTaskAttributeWithCallback invokes the ecs.DescribeTaskAttribute API asynchronously
func (client *Client) DescribeTaskAttributeWithCallback(request *DescribeTaskAttributeRequest, callback func(response *DescribeTaskAttributeResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeTaskAttributeResponse
		var err error
		defer close(result)
		response, err = client.DescribeTaskAttribute(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// DescribeTaskAttributeRequest is the request struct for api DescribeTaskAttribute
type DescribeTaskAttributeRequest struct {
	*requests.RpcRequest
	ResourceOwnerId      requests.Integer `position:"Query" name:"ResourceOwnerId"`
	TaskId               string           `position:"Query" name:"TaskId"`
	ResourceOwnerAccount string           `position:"Query" name:"ResourceOwnerAccount"`
	OwnerId              requests.Integer `position:"Query" name:"OwnerId"`
}

// DescribeTaskAttributeResponse is the response struct for api DescribeTaskAttribute
type DescribeTaskAttributeResponse struct {
	*responses.BaseResponse
	CreationTime         string                                      `json:"CreationTime" xml:"CreationTime"`
	SupportCancel        string                                      `json:"SupportCancel" xml:"SupportCancel"`
	TotalCount           int                                         `json:"TotalCount" xml:"TotalCount"`
	SuccessCount         int                                         `json:"SuccessCount" xml:"SuccessCount"`
	RegionId             string                                      `json:"RegionId" xml:"RegionId"`
	TaskAction           string                                      `json:"TaskAction" xml:"TaskAction"`
	FailedCount          int                                         `json:"FailedCount" xml:"FailedCount"`
	RequestId            string                                      `json:"RequestId" xml:"RequestId"`
	TaskStatus           string                                      `json:"TaskStatus" xml:"TaskStatus"`
	TaskProcess          string                                      `json:"TaskProcess" xml:"TaskProcess"`
	FinishedTime         string                                      `json:"FinishedTime" xml:"FinishedTime"`
	TaskId               string                                      `json:"TaskId" xml:"TaskId"`
	OperationProgressSet OperationProgressSetInDescribeTaskAttribute `json:"OperationProgressSet" xml:"OperationProgressSet"`
}

// CreateDescribeTaskAttributeRequest creates a request to invoke DescribeTaskAttribute API
func CreateDescribeTaskAttributeRequest() (request *DescribeTaskAttributeRequest) {
	request = &DescribeTaskAttributeRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Ecs", "2014-05-26", "DescribeTaskAttribute", "ecs", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDescribeTaskAttributeResponse creates a response to parse from DescribeTaskAttribute response
func CreateDescribeTaskAttributeResponse() (response *DescribeTaskAttributeResponse) {
	response = &DescribeTaskAttributeResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
