package dockerauth

import (
	"testing"
	"fmt"
	"encoding/base64"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestECRAuthProvider_extractRegistryInfo(t *testing.T) {
	registryId, registryRegion, err := extractRegistryInfo("123456789012.dkr.ecr.us-east-1.amazonaws.com")
	assert.NoError(t, err)
	assert.Equal(t, "123456789012", registryId)
	assert.Equal(t, "us-east-1", registryRegion)

	_, _, err = extractRegistryInfo("https://index.docker.io/v1")
	assert.EqualError(t, err, "\"https://index.docker.io/v1\" is not an ECR registry")

	_, _, err = extractRegistryInfo("")
	assert.EqualError(t, err, "\"\" is not an ECR registry")
}

func TestECRAuthProvider_newAuthConfiguration(t *testing.T) {
	authConf, err := newAuthConfiguration(encodeToken("foo:bar"), "https://123456789012.dkr.ecr.us-east-1.amazonaws.com")
	assert.NoError(t, err)
	assert.Equal(t, "foo", authConf.Username)
	assert.Equal(t, "bar", authConf.Password)
	assert.Equal(t, "https://123456789012.dkr.ecr.us-east-1.amazonaws.com", authConf.ServerAddress)
}

func TestECRAuthProvider_AuthConfiguration(t *testing.T) {
	svc := new(mockECR)
	svc.On("GetAuthorizationToken", newGetAuthorizationTokenInput("123456789012")).Return(newGetAuthorizationTokenOutput("123456789012", "us-east-1", "foo:bar"), nil)
	svc.On("GetAuthorizationToken", newGetAuthorizationTokenInput("123456789013")).Return(&ecr.GetAuthorizationTokenOutput{}, nil)
	provider := NewECRAuthProvider(func(region string) ECR {
		assert.Equal(t, "us-east-1", region)
		return svc
	})
	authConf, err := provider.AuthConfiguration("123456789012.dkr.ecr.us-east-1.amazonaws.com")
	assert.NoError(t, err)
	assert.Equal(t, "foo", authConf.Username)
	assert.Equal(t, "bar", authConf.Password)
	assert.Equal(t, "https://123456789012.dkr.ecr.us-east-1.amazonaws.com", authConf.ServerAddress)

	_, err = provider.AuthConfiguration("123456789013.dkr.ecr.us-east-1.amazonaws.com")
	assert.EqualError(t, err, "No ECR authorization data for \"123456789013.dkr.ecr.us-east-1.amazonaws.com\"")

	authConf, err = provider.AuthConfiguration("https://index.docker.io/v1")
	assert.NoError(t, err)
	assert.Nil(t, authConf)
}

func encodeToken(token string) string {
	return base64.StdEncoding.EncodeToString([]byte(token))
}

func newGetAuthorizationTokenInput(registryId string) *ecr.GetAuthorizationTokenInput {
	return &ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			aws.String(registryId),
		},
	}
}

func newExpiry() time.Time {
	ttl, _ := time.ParseDuration("12h")
	return time.Now().UTC().Add(ttl)
}

func newGetAuthorizationTokenOutput(registryId string, region string, unEncodedToken string) *ecr.GetAuthorizationTokenOutput {
	authData := []*ecr.AuthorizationData{
		&ecr.AuthorizationData{
			AuthorizationToken: aws.String(encodeToken(unEncodedToken)),
			ExpiresAt: aws.Time(newExpiry()),
			ProxyEndpoint: aws.String(fmt.Sprintf("https://%s.dkr.ecr.%s.amazonaws.com", registryId, region)),
		},
	}

	return &ecr.GetAuthorizationTokenOutput{
		AuthorizationData: authData,
	}
}

type mockECR struct {
	mock.Mock
}

func (m *mockECR) GetAuthorizationToken(input *ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error) {
	ret := m.Called(input)
	return ret.Get(0).(*ecr.GetAuthorizationTokenOutput), ret.Error(1)
}
