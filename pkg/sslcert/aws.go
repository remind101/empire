package sslcert

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/iam"
)

type IAMManager struct {
	iam  *iam.IAM
	path string
}

func NewIAMManager(config client.ConfigProvider, path string) *IAMManager {
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

	return *output.ServerCertificateMetadata.Arn, nil
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

	if out.ServerCertificate.ServerCertificateMetadata.Arn != nil {
		data["ARN"] = *out.ServerCertificate.ServerCertificateMetadata.Arn
	}

	return data, nil
}

func noCertificate(err error) bool {
	if err, ok := err.(awserr.Error); ok {
		if err.Code() == "NoSuchEntity" {
			return true
		}
	}

	return false
}
