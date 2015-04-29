// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// Keys represent public SSH keys associated with an account and are used to
// authorize accounts as they are performing git operations.
type Key struct {
	// comment on the key
	Comment string `json:"comment"`

	// when key was created
	CreatedAt time.Time `json:"created_at"`

	// deprecated. Please refer to 'comment' instead
	Email string `json:"email"`

	// a unique identifying string based on contents
	Fingerprint string `json:"fingerprint"`

	// unique identifier of this key
	Id string `json:"id"`

	// full public_key as uploaded
	PublicKey string `json:"public_key"`

	// when key was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new key.
//
// publicKey is the full public_key as uploaded.
func (c *Client) KeyCreate(publicKey string) (*Key, error) {
	params := struct {
		PublicKey string `json:"public_key"`
	}{
		PublicKey: publicKey,
	}
	var keyRes Key
	return &keyRes, c.Post(&keyRes, "/account/keys", params)
}

// Delete an existing key
//
// keyIdentity is the unique identifier of the Key.
func (c *Client) KeyDelete(keyIdentity string) error {
	return c.Delete("/account/keys/" + keyIdentity)
}

// Info for existing key.
//
// keyIdentity is the unique identifier of the Key.
func (c *Client) KeyInfo(keyIdentity string) (*Key, error) {
	var key Key
	return &key, c.Get(&key, "/account/keys/"+keyIdentity)
}

// List existing keys.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) KeyList(lr *ListRange) ([]Key, error) {
	req, err := c.NewRequest("GET", "/account/keys", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var keysRes []Key
	return keysRes, c.DoReq(req, &keysRes)
}
