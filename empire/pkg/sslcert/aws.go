package sslcert

import (
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
	primary, chain := SplitCertChain(cert)
	input := &iam.UploadServerCertificateInput{
		CertificateBody:       aws.String(primary),
		PrivateKey:            aws.String(key),
		ServerCertificateName: aws.String(name),
		Path: aws.String(m.path),
	}

	if len(chain) > 0 {
		input.CertificateChain = aws.String(chain)
	}

	output, err := m.iam.UploadServerCertificate(input)
	if err != nil {
		return "", err
	}

	return *output.ServerCertificateMetadata.ARN, nil
}

func (m *IAMManager) Remove(name string) error {
	_, err := m.iam.DeleteServerCertificate(&iam.DeleteServerCertificateInput{ServerCertificateName: aws.String(name)})
	if noCertificate(err) {
		return nil
	}
	return err
}

func (m *IAMManager) MetaData(name string) (map[string]string, error) {
	data := map[string]string{}
	out, err := m.iam.GetServerCertificate(&iam.GetServerCertificateInput{ServerCertificateName: aws.String(name)})
	if err != nil {
		return data, err
	}

	if out.ServerCertificate.ServerCertificateMetadata.ARN != nil {
		data["ARN"] = *out.ServerCertificate.ServerCertificateMetadata.ARN
	}

	return data, nil
}

func noCertificate(err error) bool {
	if err, ok := err.(aws.APIError); ok {
		if err.Code == "NoSuchEntity" {
			return true
		}
	}

	return false
}
