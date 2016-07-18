package customresources

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

var ctx = context.Background()

func TestWithTimeout_NoTimeout(t *testing.T) {
	m := new(mockProvisioner)
	p := WithTimeout(m, time.Second, time.Second)

	m.On("Provision", Request{}).Return("id", nil, nil)

	p.Provision(ctx, Request{})
}

func TestWithTimeout_Timeout_Cleanup(t *testing.T) {
	m := new(mockProvisioner)
	p := WithTimeout(m, time.Millisecond*500, time.Millisecond*500)

	m.On("Provision", Request{}).Return("id", nil, nil).Run(func(mock.Arguments) {
		time.Sleep(time.Millisecond * 750)
	})

	id, _, err := p.Provision(ctx, Request{})
	assert.NoError(t, err)
	assert.Equal(t, "id", id)
}

func TestWithTimeout_GraceTimeout(t *testing.T) {
	m := new(mockProvisioner)
	p := WithTimeout(m, time.Millisecond*500, time.Millisecond*500)

	m.On("Provision", Request{}).Return("id", nil, nil).Run(func(mock.Arguments) {
		time.Sleep(time.Millisecond * 1500)
	})

	_, _, err := p.Provision(ctx, Request{})
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestResponse_MarshalJSON(t *testing.T) {
	tests := map[Response]string{
		Response{
			Status: StatusSuccess,
		}: `{"Status":"SUCCESS","PhysicalResourceId":"","StackId":"","RequestId":"","LogicalResourceId":""}`,

		Response{
			Status: StatusFailed,
			Reason: "errored",
		}: `{"Status":"FAILED","Reason":"errored","PhysicalResourceId":"","StackId":"","RequestId":"","LogicalResourceId":""}`,
	}

	for resp, expected := range tests {
		raw, err := json.Marshal(resp)
		assert.NoError(t, err)

		assert.Equal(t, expected, string(raw))
	}
}

func TestResponseClient(t *testing.T) {
	c := ResponseClient{http.DefaultClient}

	response := Response{
		Status:             StatusSuccess,
		PhysicalResourceId: "9001",
		StackId:            "arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6",
		RequestId:          "daf3f3f9-79a1-4049-823e-09544e582b06",
		LogicalResourceId:  "webInstancePort",
		Data:               map[string]int64{"InstancePort": 9001},
	}

	var called bool
	check := func(w http.ResponseWriter, r *http.Request) {
		called = true

		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6|webInstancePort|daf3f3f9-79a1-4049-823e-09544e582b06", r.URL.Path)

		expectedRaw, err := json.Marshal(response)
		assert.NoError(t, err)
		raw, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, string(expectedRaw), string(raw))
	}
	s := httptest.NewServer(http.HandlerFunc(check))
	defer s.Close()

	req := Request{
		RequestType:       Create,
		ResponseURL:       s.URL + "/arn%3Aaws%3Acloudformation%3Aus-east-1%3A066251891493%3Astack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6%7CwebInstancePort%7Cdaf3f3f9-79a1-4049-823e-09544e582b06?AWSAccessKeyId=AKIAJNXHFR7P7YGKLDPQ&Expires=1461987599&Signature=EqV%2BqIUAsZPz5Q%2F%2B75Guvn%2BNREU%3D",
		StackId:           "arn:aws:cloudformation:us-east-1:066251891493:stack/foo/70213b00-0e74-11e6-b4fb-500c28680ac6",
		RequestId:         "daf3f3f9-79a1-4049-823e-09544e582b06",
		LogicalResourceId: "webInstancePort",
		ResourceType:      "Custom::InstancePort",
		ResourceProperties: map[string]interface{}{
			"ServiceToken": "arn:aws:sns:us-east-1:066251891493:empire-e01a8fac-CustomResourcesTopic-9KHPNW7WFKBD",
		},
	}

	err := c.SendResponse(req, response)
	assert.NoError(t, err)

	assert.True(t, called)
}

type mockProvisioner struct {
	mock.Mock
}

func (m *mockProvisioner) Provision(_ context.Context, req Request) (string, interface{}, error) {
	args := m.Called(req)
	return args.String(0), args.Get(1), args.Error(2)
}

func (m *mockProvisioner) Properties() interface{} {
	return nil
}

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}
