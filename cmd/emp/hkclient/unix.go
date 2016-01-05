// +build darwin freebsd linux netbsd openbsd

package hkclient

import "os"

const netrcFilename = ".netrc"

// user.Current() requires cgo and thus doesn't work with cross-compiling.
// The following is an alternative that matches how the Heroku Toolbelt
// works, though per @fdr it may not be correct for all cases (when users have
// modified their home dir).
//
// http://stackoverflow.com/questions/7922270/obtain-users-home-directory
func homePath() string {
	return os.Getenv("HOME")
}
