package empire

import (
	"fmt"
	"time"

	"github.com/remind101/empire/pkg/headerutil"
	"github.com/remind101/empire/pkg/timex"
)

// Release is a combination of a Config and a Slug, which form a deployable
// release. Releases are generally considered immutable, the only operation that
// changes a release is when altering the Quantity or Constraints inside the
// Formation.
type Release struct {
	// The application that this release relates to.
	App *App

	// A description for the release. Usually contains the reason for why
	// the release was created (e.g. deployment, config changes, etc).
	Description string

	// The time that this release was created.
	CreatedAt *time.Time
}

// BeforeCreate sets created_at before inserting.
func (r *Release) BeforeCreate() error {
	t := timex.Now()
	r.CreatedAt = &t
	return nil
}

// ReleasesQuery is a scope implementation for common things to filter releases
// by.
type ReleasesQuery struct {
	// If provided, an app to filter by.
	App *App

	// If provided, a version to filter by.
	Version *int

	// If provided, uses the limit and sorting parameters specified in the range.
	Range headerutil.Range
}

func appendMessageToDescription(main string, user *User, message string) string {
	var formatted string
	if message != "" {
		formatted = fmt.Sprintf(": '%s'", message)
	}
	return fmt.Sprintf("%s (%s%s)", main, user.Name, formatted)
}
