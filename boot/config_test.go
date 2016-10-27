package boot

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateProductionConfig(t *testing.T) {
	tests := []struct {
		config string
		result *ValidationResult
	}{
		// Invalid auth backend
		{
			config: `
[db]
url = "postgres://localhost/empire?sslmode=disable"

[server.auth]
secret = "secret"
backend = "foo"`,
			result: &ValidationResult{
				Errors: []error{
					ValueError{
						Name: "backend",
						Path: "server.auth",
						Err:  errors.New("valid values are: [fake github saml]"),
					},
				},
			},
		},

		{
			// Invalid SAML configuration.
			config: `
[db]
url = "postgres://localhost/empire?sslmode=disable"

[server.auth]
secret = "secret"
backend = "saml"

[server.auth.saml]
metadata = "https://app.onelogin.com/metadata.xml"`,
			result: &ValidationResult{
				Errors: []error{
					ValueError{
						Name: "url",
						Path: "server",
						Err:  errors.New("missing"),
					},
					ValueError{
						Name: "key",
						Path: "server.auth.saml",
						Err:  errors.New("missing"),
					},
					ValueError{
						Name: "cert",
						Path: "server.auth.saml",
						Err:  errors.New("missing"),
					},
				},
			},
		},

		{
			// Invalid GitHub configuration
			config: `
[db]
url = "postgres://localhost/empire?sslmode=disable"

[server.auth]
secret = "secret"
backend = "github"

[server.auth.github]
client_id = "<client_id>"
client_secret = "<client_secret>"`,
			result: nil,
		},
		{
			// CloudFormation scheduler missing CustomResources
			// configuration.
			config: `
[db]
url = "postgres://localhost/empire?sslmode=disable"

[server.auth]
secret = "secret"

[scheduler]
backend = "cloudformation"

[scheduler.cloudformation]
vpc_id = "vpc-d315edb4"
template_bucket = "empire-77792028-templatebucket-195oucd149ybu"
elb_private_security_group = "sg-f33ef988"
ec2_private_subnets = ["subnet-d280dfa4", "subnet-89402ad1"]
elb_public_security_group = "sg-fa3ef981"
ec2_public_subnets = ["subnet-d280dfa4", "subnet-89402ad1"]
ecs_cluster = "empire-77792028-Cluster-1CU7HL67LPPHO"
ecs_service_role = "empire-77792028-ServiceRole-1SD04YIKP9AIP"
route53_internal_hosted_zone_id = "Z185KSIEQC21FF"`,
			result: &ValidationResult{
				Errors: []error{
					ValueError{
						Name: "topic",
						Path: "cloudformation_customresources",
						Err:  errors.New("missing"),
					},
				},
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			config, err := ParseConfig(strings.NewReader(tt.config))
			if assert.NoError(t, err) {
				r := ValidateProductionConfig(config)
				if r != nil {
					t.Log(r.Error())
				}
				if tt.result == nil {
					assert.Nil(t, r)
				} else {
					assert.Equal(t, tt.result.Errors, r.Errors)
				}
			}
		})
	}
}
