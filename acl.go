package empire

import "github.com/remind101/empire/acl"

// These are primary here as examples, and for testing. Admins should define
// their own policies, and attach them to groups of users.
var (
	// A policy that allows complete administrator level access.
	AccessAdmin = acl.Policy{
		acl.Statement{
			Effect:   acl.Allow,
			Resource: []string{"*"},
			Action:   []string{"*"},
		},
	}

	// A policy that allows normal "operational" actions, like deploying,
	// changing config, scaling, etc.
	AccessOperator = acl.Policy{
		acl.Statement{
			Effect:   acl.Allow,
			Resource: []string{"*"},
			Action: []string{
				"Deploy",
				"Set",
				"Config",
				"Rollback",
				"Scale",
				"Run",
				"Restart",
				"StreamLogs",
			},
		},
	}

	// A policy that only allows Deployments.
	AccessDeployOnly = acl.Policy{
		acl.Statement{
			Effect:   acl.Allow,
			Resource: []string{"*"},
			Action: []string{
				"Deploy",
			},
		},
	}
)
