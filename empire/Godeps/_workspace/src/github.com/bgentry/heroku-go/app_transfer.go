// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// An app transfer represents a two party interaction for transferring ownership
// of an app.
type AppTransfer struct {
	// app involved in the transfer
	App struct {
		Name string `json:"name"`
		Id   string `json:"id"`
	} `json:"app"`

	// when app transfer was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of app transfer
	Id string `json:"id"`

	// identity of the owner of the transfer
	Owner struct {
		Email string `json:"email"`
		Id    string `json:"id"`
	} `json:"owner"`

	// identity of the recipient of the transfer
	Recipient struct {
		Email string `json:"email"`
		Id    string `json:"id"`
	} `json:"recipient"`

	// the current state of an app transfer
	State string `json:"state"`

	// when app transfer was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new app transfer.
//
// app is the unique identifier of app or unique name of app. recipient is the
// unique email address of account or unique identifier of an account.
func (c *Client) AppTransferCreate(app string, recipient string) (*AppTransfer, error) {
	params := struct {
		App       string `json:"app"`
		Recipient string `json:"recipient"`
	}{
		App:       app,
		Recipient: recipient,
	}
	var appTransferRes AppTransfer
	return &appTransferRes, c.Post(&appTransferRes, "/account/app-transfers", params)
}

// Delete an existing app transfer
//
// appTransferIdentity is the unique identifier of the AppTransfer.
func (c *Client) AppTransferDelete(appTransferIdentity string) error {
	return c.Delete("/account/app-transfers/" + appTransferIdentity)
}

// Info for existing app transfer.
//
// appTransferIdentity is the unique identifier of the AppTransfer.
func (c *Client) AppTransferInfo(appTransferIdentity string) (*AppTransfer, error) {
	var appTransfer AppTransfer
	return &appTransfer, c.Get(&appTransfer, "/account/app-transfers/"+appTransferIdentity)
}

// List existing apps transfers.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) AppTransferList(lr *ListRange) ([]AppTransfer, error) {
	req, err := c.NewRequest("GET", "/account/app-transfers", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var appTransfersRes []AppTransfer
	return appTransfersRes, c.DoReq(req, &appTransfersRes)
}

// Update an existing app transfer.
//
// appTransferIdentity is the unique identifier of the AppTransfer. state is the
// the current state of an app transfer.
func (c *Client) AppTransferUpdate(appTransferIdentity string, state string) (*AppTransfer, error) {
	params := struct {
		State string `json:"state"`
	}{
		State: state,
	}
	var appTransferRes AppTransfer
	return &appTransferRes, c.Patch(&appTransferRes, "/account/app-transfers/"+appTransferIdentity, params)
}
