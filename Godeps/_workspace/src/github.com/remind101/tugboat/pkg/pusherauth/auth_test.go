package pusherauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const testSecret = "7ad3773142a6692b25b8"

func TestSign(t *testing.T) {
	tests := []struct {
		socketID string
		channel  string

		signature string
	}{
		{"1234.1234", "private-foobar", "58df8b0c36d6982b82c3ecf6b4662e34fe8c25bba48f5369f135bf843651c3a4"},
	}

	for _, tt := range tests {
		sig := Sign([]byte(testSecret), tt.socketID, tt.channel)

		if got, want := sig, tt.signature; got != want {
			t.Errorf("Sign(%s, %s) => %q; want %q", tt.socketID, tt.channel, got, want)
		}
	}
}

func TestHandler(t *testing.T) {
	h := &Handler{Key: "278d425bdf160c739803", Secret: []byte(testSecret)}

	req, _ := http.NewRequest("POST", "?channel_name=private-foobar&socket_id=1234.1234", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	expected := `{"auth":"278d425bdf160c739803:58df8b0c36d6982b82c3ecf6b4662e34fe8c25bba48f5369f135bf843651c3a4"}` + "\n"

	if got, want := resp.Body.String(), expected; got != want {
		t.Fatalf("Body => %s; want %s", got, want)
	}
}
