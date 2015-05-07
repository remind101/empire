package sslcert

import (
	"fmt"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/iam"
)

type IAMManager struct {
	iam  *iam.IAM
	path string
}

func NewIAMManager(config *aws.Config, path string) *IAMManager {
	return &IAMManager{
		iam:  iam.New(config),
		path: path,
	}
}

func (m *IAMManager) Add(name string, cert string, key string) (string, error) {
	input := &iam.UploadServerCertificateInput{
		CertificateBody:       aws.String("certificateBodyType"), // TODO parse from cert chain
		PrivateKey:            aws.String(key),
		ServerCertificateName: certName(name),
		CertificateChain:      aws.String(cert),
		Path:                  aws.String(m.path),
	}
	output, err := m.iam.UploadServerCertificate(input)
	if err != nil {
		return "", err
	}

	return *output.ServerCertificateMetadata.ServerCertificateID, nil
}

func (m *IAMManager) Remove(name string) error {
	_, err := m.iam.DeleteServerCertificate(&iam.DeleteServerCertificateInput{ServerCertificateName: certName(name)})
	return err
}

func (m *IAMManager) MetaData(name string) (map[string]string, error) {
	data := map[string]string{}
	out, err := m.iam.GetServerCertificate(&iam.GetServerCertificateInput{ServerCertificateName: certName(name)})
	if err != nil {
		return data, err
	}

	if out.ServerCertificate.ServerCertificateMetadata.ARN != nil {
		data["ARN"] = *out.ServerCertificate.ServerCertificateMetadata.ARN
	}

	return data, nil
}

func certName(name string) *string {
	return aws.String(fmt.Sprintf("%sCert", name))
}
