// package pusherauth is a package for generating pusher authentication
// signatures. See https://pusher.com/docs/authenticating_users and
// https://pusher.com/docs/auth_signatures.
package pusherauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
)

// Sign generates a suitable HMAC signature of the socket and channel.
func Sign(secret []byte, socketID, channel string) string {
	m := fmt.Sprintf("%s:%s", socketID, channel)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(m))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// Handler is an http.Handler that will respond with a pusher auth string.
type Handler struct {
	// Pusher application key.
	Key string

	// Pusher secret, used to sign the auth payload.
	Secret []byte
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	socketID := r.FormValue("socket_id")
	channel := r.FormValue("channel_name")

	sig := Sign(h.Secret, socketID, channel)

	json.NewEncoder(w).Encode(map[string]string{
		"auth": fmt.Sprintf("%s:%s", h.Key, sig),
	})
}
