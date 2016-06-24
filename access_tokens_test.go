package empire

import (
	"reflect"
	"testing"
)

var testSecret = []byte("secret")

func TestAccessTokensFind(t *testing.T) {
	s := &accessTokensService{Empire: &Empire{Secret: testSecret}}

	at, err := s.AccessTokensFind("")
	if err != nil {
		t.Logf("err: %v", reflect.TypeOf(err))
		t.Fatal(err)
	}

	if at != nil {
		t.Fatal("Expected access token to be nil")
	}
}
