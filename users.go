package empire

// User represents a user of Empire.
type User struct {
	// Name is the users username.
	Name string `json:"name"`

	// GitHubToken is a GitHub access token.
	GitHubToken string `json:"-"`
}

// IsValid returns nil if the User is valid.
func (u *User) IsValid() error {
	if u.Name == "" {
		return ErrUserName
	}

	return nil
}
