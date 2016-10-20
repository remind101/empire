package onelogin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSAMLAssertionResponse_Success(t *testing.T) {
	b := `{
    "status": {
        "type": "success",
        "message": "Success",
        "error": false,
        "code": 200 
    },
   "data": "PHNhb+P..."
    }`

	var resp GenerateSAMLAssertionResponse
	err := json.Unmarshal([]byte(b), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "PHNhb+P...", resp.Data)
}

func TestGenerateSAMLAssertionResponse_MFA(t *testing.T) {
	b := `{
    "status": {
        "type": "success",
        "message": "MFA is required for this user",
        "code": 200,
        "error": false
    },
    "data": [
        {
            "state_token": "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
            "devices": [
                {
                    "device_id": 666666,
                    "device_type": "Google Authenticator"
                }
            ],
            "callback_url": "https://api.us.onelogin.com/api/1/saml_assertion/verify_factor",
            "user": {
                "lastname": "Zhang",
                "username": "hzhang123",
                "email": "hazel.zhang@onelogin.com",
                "firstname": "Hazel",
                "id": 88888888
            }
        }
    ]
}`

	var resp GenerateSAMLAssertionResponse
	err := json.Unmarshal([]byte(b), &resp)
	assert.NoError(t, err)
	data := resp.Data.([]*GenerateSAMLAssertionMFAData)
	assert.Equal(t, "5xxx604x8xx9x694xx860173xxx3x78x3x870x56", data[0].StateToken)
}
