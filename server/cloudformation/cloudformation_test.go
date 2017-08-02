package cloudformation

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/empire/pkg/cloudformation/customresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var ctx = context.Background()

func TestCustomResourceProvisioner_Handle(t *testing.T) {
	p := new(mockProvisioner)
	c := &CustomResourceProvisioner{
		Provisioners: map[string]customresources.Provisioner{
			"Custom::InstancePort": p,
		},
	}

	message := &sqs.Message{
		Body: aws.String(`{
  "Type" : "Notification",
  "MessageId" : "7c72a0bb-c6f6-536b-88b7-ef25c9c6734a",
  "TopicArn" : "arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD",
  "Subject" : "AWS CloudFormation custom resource request",
  "Message" : "{\"RequestType\":\"Create\",\"ServiceToken\":\"arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD\",\"ResponseURL\":\"https://cloudformation-custom-resource-response-useast1.s3.amazonaws.com/arn%3Aaws%3Acloudformation%3Aus-east-1%3A066251891493%3Astack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6%7CwebInstancePort%7Cdaf3f3f9-79a1-4049-823e-09544e582b06?AWSAccessKeyId=AKIAJNXHFR7P7YGKLDPQ&Expires=1461987599&Signature=EqV%2BqIUAsZPz5Q%2F%2B75Guvn%2BNREU%3D\",\"StackId\":\"arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6\",\"RequestId\":\"daf3f3f9-79a1-4049-823e-09544e582b06\",\"LogicalResourceId\":\"webInstancePort\",\"ResourceType\":\"Custom::InstancePort\",\"ResourceProperties\":{\"ServiceToken\":\"arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD\"}}",
  "Timestamp" : "2016-04-30T01:40:00.042Z",
  "SignatureVersion" : "1",
  "Signature" : "cZI/3gLQzH7hXmjh6O2FVRGf+rylVCuLieuDjqA+ptQeM+VWXptga8p7+VJGl2tgijqDLOST20ErHxMVeE3Gq5eA2zLtydJcZfbzI/jyBSdM41NalrrLsVENi1N318KJ+5eGgKB9MvUMqQb0/BrbzIEuzmbCRe3P60188J/ME/5CBsRB/jfUbr7+asN5qJIf4B/CluVfoF5n1bbBmLA5YqttisB7Y626Bvr8EM9S/NdlNHfwq3ZIA+OQkTUzVKmwQsE1h7ICNm+UQxZgca+JRuPq7QRstHeuiIjMEn7/Q4UPh2FknqSEu8vtu/kdA8oUhA5WvcN59V5kog9mo3Q1WA==",
  "SigningCertURL" : "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-bb750dd426d95ee9390147a5624348ee.pem",
  "UnsubscribeURL" : "https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD:d369ecb8-031f-40b3-bf8d-37b7cdc30fbe"
}`),
	}

	req := customresources.Request{
		RequestType:       customresources.Create,
		ResponseURL:       "https://cloudformation-custom-resource-response-useast1.s3.amazonaws.com/arn%3Aaws%3Acloudformation%3Aus-east-1%3A066251891493%3Astack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6%7CwebInstancePort%7Cdaf3f3f9-79a1-4049-823e-09544e582b06?AWSAccessKeyId=AKIAJNXHFR7P7YGKLDPQ&Expires=1461987599&Signature=EqV%2BqIUAsZPz5Q%2F%2B75Guvn%2BNREU%3D",
		StackId:           "arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6",
		RequestId:         "daf3f3f9-79a1-4049-823e-09544e582b06",
		LogicalResourceId: "webInstancePort",
		ResourceType:      "Custom::InstancePort",
		ResourceProperties: map[string]interface{}{
			"ServiceToken": "arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD",
		},
	}
	resp := customresources.Response{
		Status:             customresources.StatusSuccess,
		PhysicalResourceId: "9001",
		StackId:            "arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6",
		RequestId:          "daf3f3f9-79a1-4049-823e-09544e582b06",
		LogicalResourceId:  "webInstancePort",
		Data:               map[string]int64{"InstancePort": 9001},
	}

	c.sendResponse = checkSendResponse(t, req, resp)

	p.On("Provision", req).Return("9001", map[string]int64{"InstancePort": 9001}, nil)

	err := c.Handle(context.Background(), message)
	assert.NoError(t, err)

	p.AssertExpectations(t)
}

func TestCustomResourceProvisioner_Handle_CreateFailed(t *testing.T) {
	p := new(mockProvisioner)
	c := &CustomResourceProvisioner{
		Provisioners: map[string]customresources.Provisioner{
			"Custom::InstancePort": p,
		},
	}

	message := &sqs.Message{
		Body: aws.String(`{
  "Type" : "Notification",
  "MessageId" : "7c72a0bb-c6f6-536b-88b7-ef25c9c6734a",
  "TopicArn" : "arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD",
  "Subject" : "AWS CloudFormation custom resource request",
  "Message" : "{\"RequestType\":\"Create\",\"ServiceToken\":\"arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD\",\"ResponseURL\":\"https://cloudformation-custom-resource-response-useast1.s3.amazonaws.com/arn%3Aaws%3Acloudformation%3Aus-east-1%3A066251891493%3Astack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6%7CwebInstancePort%7Cdaf3f3f9-79a1-4049-823e-09544e582b06?AWSAccessKeyId=AKIAJNXHFR7P7YGKLDPQ&Expires=1461987599&Signature=EqV%2BqIUAsZPz5Q%2F%2B75Guvn%2BNREU%3D\",\"StackId\":\"arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6\",\"RequestId\":\"daf3f3f9-79a1-4049-823e-09544e582b06\",\"LogicalResourceId\":\"webInstancePort\",\"ResourceType\":\"Custom::InstancePort\",\"ResourceProperties\":{\"ServiceToken\":\"arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD\"}}",
  "Timestamp" : "2016-04-30T01:40:00.042Z",
  "SignatureVersion" : "1",
  "Signature" : "cZI/3gLQzH7hXmjh6O2FVRGf+rylVCuLieuDjqA+ptQeM+VWXptga8p7+VJGl2tgijqDLOST20ErHxMVeE3Gq5eA2zLtydJcZfbzI/jyBSdM41NalrrLsVENi1N318KJ+5eGgKB9MvUMqQb0/BrbzIEuzmbCRe3P60188J/ME/5CBsRB/jfUbr7+asN5qJIf4B/CluVfoF5n1bbBmLA5YqttisB7Y626Bvr8EM9S/NdlNHfwq3ZIA+OQkTUzVKmwQsE1h7ICNm+UQxZgca+JRuPq7QRstHeuiIjMEn7/Q4UPh2FknqSEu8vtu/kdA8oUhA5WvcN59V5kog9mo3Q1WA==",
  "SigningCertURL" : "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-bb750dd426d95ee9390147a5624348ee.pem",
  "UnsubscribeURL" : "https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD:d369ecb8-031f-40b3-bf8d-37b7cdc30fbe"
}`),
	}

	req := customresources.Request{
		RequestType:       customresources.Create,
		ResponseURL:       "https://cloudformation-custom-resource-response-useast1.s3.amazonaws.com/arn%3Aaws%3Acloudformation%3Aus-east-1%3A066251891493%3Astack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6%7CwebInstancePort%7Cdaf3f3f9-79a1-4049-823e-09544e582b06?AWSAccessKeyId=AKIAJNXHFR7P7YGKLDPQ&Expires=1461987599&Signature=EqV%2BqIUAsZPz5Q%2F%2B75Guvn%2BNREU%3D",
		StackId:           "arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6",
		RequestId:         "daf3f3f9-79a1-4049-823e-09544e582b06",
		LogicalResourceId: "webInstancePort",
		ResourceType:      "Custom::InstancePort",
		ResourceProperties: map[string]interface{}{
			"ServiceToken": "arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD",
		},
	}
	resp := customresources.Response{
		Status:             customresources.StatusFailed,
		PhysicalResourceId: "failed/Create",
		Reason:             "pq: unique violation",
		StackId:            "arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6",
		RequestId:          "daf3f3f9-79a1-4049-823e-09544e582b06",
		LogicalResourceId:  "webInstancePort",
		Data:               nil,
	}

	c.sendResponse = checkSendResponse(t, req, resp)

	p.On("Provision", req).Return("", nil, errors.New("pq: unique violation"))

	err := c.Handle(context.Background(), message)
	assert.NoError(t, err)

	p.AssertExpectations(t)
}

func TestCustomResourceProvisioner_Handle_DeleteFailedResource(t *testing.T) {
	p := new(mockProvisioner)
	c := &CustomResourceProvisioner{
		Provisioners: map[string]customresources.Provisioner{
			"Custom::InstancePort": p,
		},
	}

	message := &sqs.Message{
		Body: aws.String(`{
  "Type" : "Notification",
  "MessageId" : "7c72a0bb-c6f6-536b-88b7-ef25c9c6734a",
  "TopicArn" : "arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD",
  "Subject" : "AWS CloudFormation custom resource request",
  "Message" : "{\"RequestType\":\"Delete\",\"PhysicalResourceId\":\"failed/Create\",\"ServiceToken\":\"arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD\",\"ResponseURL\":\"https://cloudformation-custom-resource-response-useast1.s3.amazonaws.com/arn%3Aaws%3Acloudformation%3Aus-east-1%3A066251891493%3Astack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6%7CwebInstancePort%7Cdaf3f3f9-79a1-4049-823e-09544e582b06?AWSAccessKeyId=AKIAJNXHFR7P7YGKLDPQ&Expires=1461987599&Signature=EqV%2BqIUAsZPz5Q%2F%2B75Guvn%2BNREU%3D\",\"StackId\":\"arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6\",\"RequestId\":\"daf3f3f9-79a1-4049-823e-09544e582b06\",\"LogicalResourceId\":\"webInstancePort\",\"ResourceType\":\"Custom::InstancePort\",\"ResourceProperties\":{\"ServiceToken\":\"arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD\"}}",
  "Timestamp" : "2016-04-30T01:40:00.042Z",
  "SignatureVersion" : "1",
  "Signature" : "cZI/3gLQzH7hXmjh6O2FVRGf+rylVCuLieuDjqA+ptQeM+VWXptga8p7+VJGl2tgijqDLOST20ErHxMVeE3Gq5eA2zLtydJcZfbzI/jyBSdM41NalrrLsVENi1N318KJ+5eGgKB9MvUMqQb0/BrbzIEuzmbCRe3P60188J/ME/5CBsRB/jfUbr7+asN5qJIf4B/CluVfoF5n1bbBmLA5YqttisB7Y626Bvr8EM9S/NdlNHfwq3ZIA+OQkTUzVKmwQsE1h7ICNm+UQxZgca+JRuPq7QRstHeuiIjMEn7/Q4UPh2FknqSEu8vtu/kdA8oUhA5WvcN59V5kog9mo3Q1WA==",
  "SigningCertURL" : "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-bb750dd426d95ee9390147a5624348ee.pem",
  "UnsubscribeURL" : "https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD:d369ecb8-031f-40b3-bf8d-37b7cdc30fbe"
}`),
	}

	req := customresources.Request{
		RequestType:        customresources.Delete,
		PhysicalResourceId: "failed/Create",
		ResponseURL:        "https://cloudformation-custom-resource-response-useast1.s3.amazonaws.com/arn%3Aaws%3Acloudformation%3Aus-east-1%3A066251891493%3Astack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6%7CwebInstancePort%7Cdaf3f3f9-79a1-4049-823e-09544e582b06?AWSAccessKeyId=AKIAJNXHFR7P7YGKLDPQ&Expires=1461987599&Signature=EqV%2BqIUAsZPz5Q%2F%2B75Guvn%2BNREU%3D",
		StackId:            "arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6",
		RequestId:          "daf3f3f9-79a1-4049-823e-09544e582b06",
		LogicalResourceId:  "webInstancePort",
		ResourceType:       "Custom::InstancePort",
		ResourceProperties: map[string]interface{}{
			"ServiceToken": "arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD",
		},
	}
	resp := customresources.Response{
		Status:             customresources.StatusSuccess,
		PhysicalResourceId: "failed/Create",
		StackId:            "arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6",
		RequestId:          "daf3f3f9-79a1-4049-823e-09544e582b06",
		LogicalResourceId:  "webInstancePort",
		Data:               nil,
	}

	c.sendResponse = checkSendResponse(t, req, resp)

	err := c.Handle(context.Background(), message)
	assert.NoError(t, err)

	p.AssertExpectations(t)
}

// checkSendResponse returns a sendResponse func that checks that the req and
// response match the expected values.
func checkSendResponse(t testing.TB, expectedReq customresources.Request, expectedResponse customresources.Response) func(customresources.Request, customresources.Response) error {
	return func(req customresources.Request, response customresources.Response) error {
		assert.Equal(t, expectedReq, req)
		assert.Equal(t, expectedResponse, response)
		return nil
	}
}

type mockProvisioner struct {
	mock.Mock
}

func (m *mockProvisioner) Provision(_ context.Context, req customresources.Request) (string, interface{}, error) {
	args := m.Called(req)
	return args.String(0), args.Get(1), args.Error(2)
}

func (m *mockProvisioner) Properties() interface{} {
	return nil
}

type mockSQSClient struct {
	mock.Mock
}

func (m *mockSQSClient) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (m *mockSQSClient) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sqs.DeleteMessageOutput), args.Error(1)
}

func (m *mockSQSClient) ChangeMessageVisibility(input *sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*sqs.ChangeMessageVisibilityOutput), args.Error(1)
}
