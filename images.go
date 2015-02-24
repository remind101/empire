package empire

// Image represents a container image, which is tied to a repository.
type Image struct {
	Repo Repo   `json:"repo"`
	ID   string `json:"id"`
}
