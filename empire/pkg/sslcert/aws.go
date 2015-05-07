package sslcert

import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/iam"
)

type IAMManager struct {
	iam *iam.IAM
}

func NewIAMManager(config *aws.Config) *IAMManager {
	return &IAMManager{
		iam: iam.New(config),
	}
}

func (m *IAMManager) Add(name string, cert string, key string) (string, error) {
	return "", nil
}

func (m *IAMManager) Remove(id string) error {
	return nil
}

func (m *IAMManager) MetaData(id string) (map[string]string, error) {
	return map[string]string{}, nil
}
