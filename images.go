package empire

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// Image represents a container image, which is tied to a repository.
type Image struct {
	ID   string `json:"id"`
	Repo Repo   `json:"repo"`
}

func (i Image) String() string {
	return encodeImage(i)
}

// Scan implements the sql.Scanner interface.
func (i *Image) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*i = decodeImage(string(src))
	}

	return nil
}

// Value implements the driver.Value interface.
func (i Image) Value() (driver.Value, error) {
	return driver.Value(i.String()), nil
}

func encodeImage(i Image) string {
	return fmt.Sprintf("%s:%s", i.Repo, i.ID)
}

func decodeImage(s string) Image {
	c := strings.Split(s, ":")
	return Image{
		Repo: Repo(c[0]),
		ID:   c[1],
	}
}
