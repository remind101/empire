// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

// Get config-vars for app.
//
// appIdentity is the unique identifier of the ConfigVar's App.
func (c *Client) ConfigVarInfo(appIdentity string) (map[string]string, error) {
	var configVar map[string]string
	return configVar, c.Get(&configVar, "/apps/"+appIdentity+"/config-vars")
}

// Update config-vars for app. You can update existing config-vars by setting
// them again, and remove by setting it to nil.
//
// appIdentity is the unique identifier of the ConfigVar's App. options is the
// hash of config changes – update values or delete by seting it to nil.
func (c *Client) ConfigVarUpdate(appIdentity string, options map[string]*string) (map[string]string, error) {
	var configVarRes map[string]string
	return configVarRes, c.Patch(&configVarRes, "/apps/"+appIdentity+"/config-vars", options)
}
