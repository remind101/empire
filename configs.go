package empire

import (
	"fmt"
	"sort"
	"strings"
)

// newEnvironment builds a new set of environment variables.
func newConfig(env map[string]string, vars Vars) map[string]string {
	return mergeVars(env, vars)
}

// Vars represents a variable -> value mapping.
type Vars map[string]*string

// mergeVars copies all of the vars from a, and merges b into them, returning a
// new Vars.
func mergeVars(old map[string]string, new Vars) map[string]string {
	vars := make(map[string]string)

	for n, v := range old {
		vars[n] = v
	}

	for n, v := range new {
		if v == nil {
			delete(vars, n)
		} else {
			vars[n] = *v
		}
	}

	return vars
}

// configsApplyReleaseDesc formats a release description based on the config variables
// being applied.
func configsApplyReleaseDesc(opts SetOpts) string {
	vars := opts.Vars
	verb := "Set"
	plural := ""
	if len(vars) > 1 {
		plural = "s"
	}

	keys := make(sort.StringSlice, 0, len(vars))
	for k, v := range vars {
		keys = append(keys, string(k))
		if v == nil {
			verb = "Unset"
		}
	}
	keys.Sort()
	desc := fmt.Sprintf("%s %s config var%s", verb, strings.Join(keys, ", "), plural)
	return appendMessageToDescription(desc, opts.User, opts.Message)
}
