package events

type Membership struct {
	Action string `json:"action"`
	Member struct {
		AvatarURL         string `json:"avatar_url"`
		EventsURL         string `json:"events_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		GravatarID        string `json:"gravatar_id"`
		HTMLURL           string `json:"html_url"`
		ID                int    `json:"id"`
		Login             string `json:"login"`
		OrganizationsURL  string `json:"organizations_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		ReposURL          string `json:"repos_url"`
		SiteAdmin         bool   `json:"site_admin"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		Type              string `json:"type"`
		URL               string `json:"url"`
	} `json:"member"`
	Organization struct {
		AvatarURL        string `json:"avatar_url"`
		EventsURL        string `json:"events_url"`
		ID               int    `json:"id"`
		Login            string `json:"login"`
		MembersURL       string `json:"members_url"`
		PublicMembersURL string `json:"public_members_url"`
		ReposURL         string `json:"repos_url"`
		URL              string `json:"url"`
	} `json:"organization"`
	Scope  string `json:"scope"`
	Sender struct {
		AvatarURL         string `json:"avatar_url"`
		EventsURL         string `json:"events_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		GravatarID        string `json:"gravatar_id"`
		HTMLURL           string `json:"html_url"`
		ID                int    `json:"id"`
		Login             string `json:"login"`
		OrganizationsURL  string `json:"organizations_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		ReposURL          string `json:"repos_url"`
		SiteAdmin         bool   `json:"site_admin"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		Type              string `json:"type"`
		URL               string `json:"url"`
	} `json:"sender"`
	Team struct {
		ID              int    `json:"id"`
		MembersURL      string `json:"members_url"`
		Name            string `json:"name"`
		Permission      string `json:"permission"`
		RepositoriesURL string `json:"repositories_url"`
		Slug            string `json:"slug"`
		URL             string `json:"url"`
	} `json:"team"`
}
