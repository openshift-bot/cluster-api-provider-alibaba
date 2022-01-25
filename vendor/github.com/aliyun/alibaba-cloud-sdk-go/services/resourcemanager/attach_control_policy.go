package resourcemanager

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

// AttachControlPolicy invokes the resourcemanager.AttachControlPolicy API synchronously
func (client *Client) AttachControlPolicy(request *AttachControlPolicyRequest) (response *AttachControlPolicyResponse, err error) {
	response = CreateAttachControlPolicyResponse()
	err = client.DoAction(request, response)
	return
}

// AttachControlPolicyWithChan invokes the resourcemanager.AttachControlPolicy API asynchronously
func (client *Client) AttachControlPolicyWithChan(request *AttachControlPolicyRequest) (<-chan *AttachControlPolicyResponse, <-chan error) {
	responseChan := make(chan *AttachControlPolicyResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.AttachControlPolicy(request)
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

// AttachControlPolicyWithCallback invokes the resourcemanager.AttachControlPolicy API asynchronously
func (client *Client) AttachControlPolicyWithCallback(request *AttachControlPolicyRequest, callback func(response *AttachControlPolicyResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *AttachControlPolicyResponse
		var err error
		defer close(result)
		response, err = client.AttachControlPolicy(request)
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

// AttachControlPolicyRequest is the request struct for api AttachControlPolicy
type AttachControlPolicyRequest struct {
	*requests.RpcRequest
	TargetId string `position:"Query" name:"TargetId"`
	PolicyId string `position:"Query" name:"PolicyId"`
}

// AttachControlPolicyResponse is the response struct for api AttachControlPolicy
type AttachControlPolicyResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
}

// CreateAttachControlPolicyRequest creates a request to invoke AttachControlPolicy API
func CreateAttachControlPolicyRequest() (request *AttachControlPolicyRequest) {
	request = &AttachControlPolicyRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("ResourceManager", "2020-03-31", "AttachControlPolicy", "", "")
	request.Method = requests.POST
	return
}

// CreateAttachControlPolicyResponse creates a response to parse from AttachControlPolicy response
func CreateAttachControlPolicyResponse() (response *AttachControlPolicyResponse) {
	response = &AttachControlPolicyResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
