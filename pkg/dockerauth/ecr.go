package dockerauth

import (
	"fmt"
	"strings"
	"regexp"
	"encoding/base64"

	"github.com/fsouza/go-dockerclient"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
)

var ecrRegistryExp = regexp.MustCompile("^([0-9]{12})\\.dkr\\.ecr\\.(.+?)\\.amazonaws\\.com$")

type ECR interface {
	GetAuthorizationToken(*ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error)
}

type ecrAuthProvider struct {
	svc ECR
}

func NewECRAuthProvider(svc ECR) *ecrAuthProvider {
	return &ecrAuthProvider{
		svc: svc,
	}
}

func (p *ecrAuthProvider) AuthConfiguration(registry string) (*docker.AuthConfiguration, error) {
	registryId, err := extractRegistryId(registry)
	if err != nil {
		return nil, nil
	}

	input := &ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			aws.String(registryId),
		},
	}

	output, err := p.svc.GetAuthorizationToken(input)
	if err != nil {
		return nil, err
	}

	if len(output.AuthorizationData) == 0 {
		return nil, fmt.Errorf("No ECR authorization data for %q", registry)
	}

	authData := output.AuthorizationData[0]
	return newAuthConfiguration(aws.StringValue(authData.AuthorizationToken), aws.StringValue(authData.ProxyEndpoint))
}

func extractRegistryId(registry string) (string, error) {
	matches := ecrRegistryExp.FindStringSubmatch(registry)
	if len(matches) == 0 {
		return "", fmt.Errorf("%q is not an ECR registry", registry)
	}

	return matches[1], nil
}

func newAuthConfiguration(encodedToken string, endpoint string) (*docker.AuthConfiguration, error) {
	decodedToken, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return nil, err
	}

	usernamePassword := strings.SplitN(string(decodedToken), ":", 2)
	return &docker.AuthConfiguration{
		Username: usernamePassword[0],
		Password: usernamePassword[1],
		ServerAddress: endpoint,
	}, nil
}
