// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

// Rate Limit represents the number of request tokens each account holds.
// Requests to this endpoint do not count towards the rate limit.
type RateLimit struct {
	// allowed requests remaining in current interval
	Remaining int `json:"remaining"`
}

// Info for rate limits.
func (c *Client) RateLimitInfo() (*RateLimit, error) {
	var rateLimit RateLimit
	return &rateLimit, c.Get(&rateLimit, "/account/rate-limits")
}
