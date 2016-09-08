package github

import (
	"github.com/remind101/empire"
	"github.com/remind101/empire/server/acl"
)

type TeamACL struct {
	// Maps a team name to a set of policies to use to determine of the
	// request should be allowed.
	policies map[string]acl.Policies
}

// Returns a set of policies that should be used when determining if the user
// has access to perform an action.
func (l *TeamACL) Policies(user *empire.User) acl.Policies {
	var policies acl.Policies
	// Find all teams this user is a member of.
	// Merge the policies for each team.
	return policies
}
