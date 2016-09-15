package github

import (
	"github.com/remind101/empire"
	"github.com/remind101/empire/acl"
)

type TeamACL struct {
	// Maps a team name to a set of policies to use to determine of the
	// request should be allowed.
	policy map[string]acl.Policy
}

// Returns a set of policies that should be used when determining if the user
// has access to perform an action.
func (l *TeamACL) Policy(user *empire.User) acl.Policy {
	var policy acl.Policy
	// Find all teams this user is a member of.
	// Merge the policies for each team.
	return policy
}
