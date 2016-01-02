package pusher

import (
	"testing"
)

func TestSign(t *testing.T) {
	signature := &Signature{
		"key",
		"secret",
		"POST",
		"/some/path",
		"1234",
		"1.0",
		[]byte("content"),
		map[string]string{"query": "params", "go": "here"},
	}

	expected := "5da41b658c67bb135898072d6d325e7a98e5f790d9c7c70cc5e210173d81be52"
	sig := signature.Sign()
	if expected != sig {
		t.Errorf("Sign(): Expected %s, got %s", expected, sig)
	}
}

func TestEncodedQuery(t *testing.T) {
	signature := &Signature{
		"key",
		"secret",
		"POST",
		"/some/path",
		"1234",
		"1.0",
		[]byte("content"),
		map[string]string{"query": "params", "go": "here"},
	}

	expected := "auth_key=key&auth_signature=5da41b658c67bb135898072d6d325e7a98e5f790d9c7c70cc5e210173d81be52&auth_timestamp=1234&auth_version=1.0&body_md5=9a0364b9e99bb480dd25e1f0284c8555&go=here&query=params"
	encodedQuery := signature.EncodedQuery()
	if expected != encodedQuery {
		t.Errorf("EncodedQuery(): Expected %s, got %s", expected, encodedQuery)
	}
}
